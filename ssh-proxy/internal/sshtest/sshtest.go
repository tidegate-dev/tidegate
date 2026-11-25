// Package sshtest runs an SSH host inside a test, and makes SSH keys for
// tests.
package sshtest

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/The127/go-random"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/phemera-dev/phemera/internal/secret"
)

// Host is an SSH server on a local port. One user logs in to it with a
// password. It has an ECDSA and an Ed25519 host key, like a host with
// several host keys.
//
// A shell writes back what it reads. A command does this:
//
//   - "false" ends with exit status 1.
//   - "stderr TEXT" writes TEXT and a line break to standard error.
//   - "signal" ends like a process that the signal TERM killed.
//   - "request TYPE" sends the client a channel request of that type and
//     writes the client's answer, "true" or "false", and a line break.
//   - "wait" runs until the client closes the channel.
//   - "eof" ends its output and then runs until the client closes the
//     channel.
//   - "stall" runs until the test ends and reads no further request.
//   - "hang up" closes the whole connection.
//   - Every other command is written back, followed by a line break.
type Host struct {
	// Address is the IP address that the server listens on.
	Address string
	Port    uint16
	// HostKey is the server's Ed25519 public key as the key type and the key
	// in base64, like "ssh-ed25519 AAAAC3...".
	HostKey string
	// SessionEnded gets a value each time a session on the server ends.
	SessionEnded chan struct{}

	stopped chan struct{}

	mu          sync.Mutex
	requests    []string
	connections []net.Conn
}

// Start runs an SSH server until the test ends. userName logs in to it with
// password.
func Start(t testing.TB, userName string, password secret.String) *Host {
	t.Helper()

	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), random.NewCryptoRandom().Reader())
	require.NoError(t, err)

	ecdsaSigner, err := ssh.NewSignerFromKey(ecdsaKey)
	require.NoError(t, err)

	ed25519Signer := NewSigner(t)

	config := &ssh.ServerConfig{
		PasswordCallback: func(meta ssh.ConnMetadata, given []byte) (*ssh.Permissions, error) {
			if meta.User() != userName || string(given) != password.Reveal() {
				return nil, errors.New("wrong user name or password")
			}

			return &ssh.Permissions{}, nil
		},
	}
	config.AddHostKey(ecdsaSigner)
	config.AddHostKey(ed25519Signer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	address := listener.Addr().(*net.TCPAddr).AddrPort()
	host := &Host{
		Address:      address.Addr().String(),
		Port:         address.Port(),
		HostKey:      AuthorizedKey(ed25519Signer.PublicKey()),
		SessionEnded: make(chan struct{}, 16),
		stopped:      make(chan struct{}),
	}

	t.Cleanup(func() {
		close(host.stopped)

		_ = listener.Close()

		host.mu.Lock()
		defer host.mu.Unlock()

		for _, connection := range host.connections {
			_ = connection.Close()
		}
	})

	go host.accept(listener, config)

	return host
}

// Requests returns the channel requests that the server got, in the order
// they came. An entry is the request's type, and "env NAME" for a variable.
func (h *Host) Requests() []string {
	h.mu.Lock()
	defer h.mu.Unlock()

	return append([]string(nil), h.requests...)
}

// NewSigner returns a new Ed25519 key.
func NewSigner(t testing.TB) ssh.Signer {
	t.Helper()

	_, private, err := ed25519.GenerateKey(random.NewCryptoRandom().Reader())
	require.NoError(t, err)

	signer, err := ssh.NewSignerFromKey(private)
	require.NoError(t, err)

	return signer
}

// AuthorizedKey returns key as the key type and the key in base64, like
// "ssh-ed25519 AAAAC3...".
func AuthorizedKey(key ssh.PublicKey) string {
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
}

func (h *Host) accept(listener net.Listener, config *ssh.ServerConfig) {
	for {
		connection, err := listener.Accept()
		if err != nil {
			return
		}

		h.mu.Lock()
		h.connections = append(h.connections, connection)
		h.mu.Unlock()

		go h.serve(connection, config)
	}
}

func (h *Host) serve(connection net.Conn, config *ssh.ServerConfig) {
	serverConn, channels, requests, err := ssh.NewServerConn(connection, config)
	if err != nil {
		_ = connection.Close()

		return
	}

	defer func() {
		_ = serverConn.Close()
	}()

	go ssh.DiscardRequests(requests)

	for newChannel := range channels {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "only sessions")

			continue
		}

		channel, channelRequests, err := newChannel.Accept()
		if err != nil {
			continue
		}

		go h.session(channel, channelRequests, func() { _ = serverConn.Close() })
	}
}

func (h *Host) session(channel ssh.Channel, requests <-chan *ssh.Request, hangUp func()) {
	defer func() {
		_ = channel.Close()

		select {
		case h.SessionEnded <- struct{}{}:
		default:
		}
	}()

	for request := range requests {
		switch request.Type {
		case "env":
			var variable struct{ Name, Value string }

			_ = ssh.Unmarshal(request.Payload, &variable)

			h.record("env " + variable.Name)
			reply(request, true)
		case "pty-req", "window-change":
			h.record(request.Type)
			reply(request, true)
		case "exec":
			var exec struct{ Command string }

			_ = ssh.Unmarshal(request.Payload, &exec)

			h.record(request.Type)
			reply(request, true)

			switch exec.Command {
			case "wait":
				continue
			case "eof":
				_ = channel.CloseWrite()

				continue
			case "stall":
				<-h.stopped

				return
			}

			run(channel, exec.Command, hangUp)

			return
		case "shell":
			h.record(request.Type)
			reply(request, true)

			_, _ = io.Copy(channel, channel)

			exit(channel, 0)

			return
		default:
			h.record(request.Type)
			reply(request, false)
		}
	}
}

func run(channel ssh.Channel, command string, hangUp func()) {
	switch {
	case command == "false":
		exit(channel, 1)
	case command == "signal":
		_, _ = channel.SendRequest("exit-signal", false, ssh.Marshal(struct {
			Signal     string
			CoreDumped bool
			Error      string
			Lang       string
		}{Signal: "TERM"}))
	case command == "hang up":
		hangUp()
	case strings.HasPrefix(command, "stderr "):
		_, _ = io.WriteString(channel.Stderr(), strings.TrimPrefix(command, "stderr ")+"\n")

		exit(channel, 0)
	case strings.HasPrefix(command, "request "):
		answer, _ := channel.SendRequest(strings.TrimPrefix(command, "request "), true, nil)

		_, _ = io.WriteString(channel, strconv.FormatBool(answer)+"\n")

		exit(channel, 0)
	default:
		_, _ = io.WriteString(channel, command+"\n")

		exit(channel, 0)
	}
}

func (h *Host) record(request string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.requests = append(h.requests, request)
}

func reply(request *ssh.Request, ok bool) {
	if request.WantReply {
		_ = request.Reply(ok, nil)
	}
}

func exit(channel ssh.Channel, status uint32) {
	_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{status}))
}
