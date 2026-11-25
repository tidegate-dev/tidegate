// Package target connects the ssh-proxy to the SSH host of a session.
package target

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/tidegate-dev/tidegate/internal/secret"
	"github.com/tidegate-dev/tidegate/ssh-proxy/internal/authority"
)

const connectTimeout = 10 * time.Second

// ErrHostKeyMismatch means that the host presented another key than the one
// the authority expects. The ssh-proxy sent the host nothing to log in with.
var ErrHostKeyMismatch = errors.New("the host presented another key than the expected host key")

// CredentialIssuer is the call to the authority that returns what the
// ssh-proxy logs in to a host with.
type CredentialIssuer interface {
	// IssueSessionCredential returns the password to log in to the session's
	// host with. observedHostKey is the public key that the host presented,
	// as the key type and the key in base64.
	IssueSessionCredential(ctx context.Context, sessionID authority.SessionID, observedHostKey string) (secret.String, error)
}

// Connect opens an SSH connection to the host of session and logs in. It
// asks credentials for the password only after the host presented the
// expected host key. If the host presented another key, the error wraps
// ErrHostKeyMismatch.
func Connect(ctx context.Context, credentials CredentialIssuer, session authority.PreparedSession) (*ssh.Client, error) {
	expected, _, _, _, err := ssh.ParseAuthorizedKey([]byte(session.HostKey))
	if err != nil {
		return nil, fmt.Errorf("reading the expected host key: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	var observedHostKey string

	config := &ssh.ClientConfig{
		User:              session.UserName,
		HostKeyAlgorithms: algorithmsOf(expected),
		HostKeyCallback: func(_ string, _ net.Addr, presented ssh.PublicKey) error {
			if !bytes.Equal(presented.Marshal(), expected.Marshal()) {
				return ErrHostKeyMismatch
			}

			observedHostKey = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(presented)))

			return nil
		},
		Auth: []ssh.AuthMethod{ssh.PasswordCallback(func() (string, error) {
			if observedHostKey == "" {
				return "", ErrHostKeyMismatch
			}

			password, err := credentials.IssueSessionCredential(ctx, session.SessionID, observedHostKey)
			if err != nil {
				return "", fmt.Errorf("asking the authority for the password: %w", err)
			}

			return password.Reveal(), nil
		})},
	}

	address := net.JoinHostPort(session.Host, strconv.Itoa(int(session.Port)))

	var dialer net.Dialer

	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("connecting to the host: %w", err)
	}

	// The SSH library takes no context, so closing the connection is what
	// ends a handshake with a host that stops answering.
	stopClosing := context.AfterFunc(ctx, func() {
		_ = connection.Close()
	})

	clientConn, channels, requests, err := ssh.NewClientConn(connection, address, config)
	if err != nil {
		_ = connection.Close()

		return nil, fmt.Errorf("logging in to the host: %w", err)
	}

	if !stopClosing() {
		_ = clientConn.Close()

		return nil, fmt.Errorf("logging in to the host: %w", context.Cause(ctx))
	}

	return ssh.NewClient(clientConn, channels, requests), nil
}

// A host with several host keys presents the one that the client asks for.
func algorithmsOf(key ssh.PublicKey) []string {
	if key.Type() == ssh.KeyAlgoRSA {
		return []string{ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSASHA256}
	}

	return []string{key.Type()}
}
