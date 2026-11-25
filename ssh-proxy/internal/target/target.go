// Package target connects the ssh-proxy to an SSH host.
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
)

const connectTimeout = 10 * time.Second

// ErrHostKeyMismatch means that the host presented another key than the
// expected one. The ssh-proxy sent the host nothing to log in with.
var ErrHostKeyMismatch = errors.New("the host presented another key than the expected host key")

var errNoHostKey = errors.New("the host asked for the password before it presented a key")

// Host is an SSH host and what the ssh-proxy needs to log in to it, except
// the password.
type Host struct {
	// Address is the host's DNS name or IP address, without the port.
	Address string
	Port    uint16
	// UserName is the user that the ssh-proxy logs in as.
	UserName string
	// HostKey is the public key that the host has to present, as the key
	// type and the key in base64, like "ssh-ed25519 AAAAC3...".
	HostKey string
}

// IssuePassword returns the password of a host's user. observedHostKey is the
// public key that the host presented, in the form of Host.HostKey.
type IssuePassword func(ctx context.Context, observedHostKey string) (secret.String, error)

// Connect opens an SSH connection to host and logs in. It calls
// issuePassword only after the host presented the expected host key. If the
// host presented another key, the error wraps ErrHostKeyMismatch.
func Connect(ctx context.Context, host Host, issuePassword IssuePassword) (*ssh.Client, error) {
	expectedHostKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(host.HostKey))
	if err != nil {
		return nil, fmt.Errorf("reading the expected host key: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	var observedHostKey string

	config := &ssh.ClientConfig{
		// The library's defaults still accept SHA-1 in key exchanges, for old hosts.
		Config:            ssh.Config{KeyExchanges: ssh.SupportedAlgorithms().KeyExchanges},
		User:              host.UserName,
		HostKeyAlgorithms: algorithmsOf(expectedHostKey),
		HostKeyCallback: func(_ string, _ net.Addr, presented ssh.PublicKey) error {
			if !bytes.Equal(presented.Marshal(), expectedHostKey.Marshal()) {
				return ErrHostKeyMismatch
			}

			// A host can start another key exchange while the login reads the variable.
			if observedHostKey == "" {
				observedHostKey = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(presented)))
			}

			return nil
		},
		Auth: []ssh.AuthMethod{ssh.PasswordCallback(func() (string, error) {
			if observedHostKey == "" {
				return "", errNoHostKey
			}

			password, err := issuePassword(ctx, observedHostKey)
			if err != nil {
				return "", fmt.Errorf("getting the password: %w", err)
			}

			return password.Reveal(), nil
		})},
	}

	address := net.JoinHostPort(host.Address, strconv.Itoa(int(host.Port)))

	var dialer net.Dialer

	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("connecting to the host: %w", err)
	}

	// The SSH library takes no context.
	stopClosing := context.AfterFunc(ctx, func() {
		_ = connection.Close()
	})

	clientConn, channels, requests, err := ssh.NewClientConn(connection, address, config)

	if !stopClosing() {
		_ = connection.Close()

		return nil, fmt.Errorf("logging in to the host: %w", context.Cause(ctx))
	}

	if err != nil {
		_ = connection.Close()

		return nil, fmt.Errorf("logging in to the host: %w", err)
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
