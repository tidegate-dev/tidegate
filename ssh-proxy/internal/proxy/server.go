// Package proxy serves SSH clients and relays their sessions to the SSH host
// that the authority names for their ticket.
package proxy

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/The127/go-random"
	"golang.org/x/crypto/ssh"

	"github.com/tidegate-dev/tidegate/internal/secret"
	"github.com/tidegate-dev/tidegate/ssh-proxy/internal/authority"
	"github.com/tidegate-dev/tidegate/ssh-proxy/internal/target"
	"github.com/tidegate-dev/tidegate/ssh-proxy/internal/ticket"
)

const handshakeTimeout = 30 * time.Second

// Server accepts connections that start with a preamble naming a ticket,
// followed by an SSH connection.
type Server struct {
	hostKey  ssh.Signer
	tickets  ticket.Verifier
	sessions authority.Sessions
}

// NewServer returns a server that presents hostKey to its clients, checks
// their tickets with tickets and asks sessions for the host of a ticket.
func NewServer(hostKey ssh.Signer, tickets ticket.Verifier, sessions authority.Sessions) *Server {
	return &Server{hostKey: hostKey, tickets: tickets, sessions: sessions}
}

// GenerateHostKey returns a new Ed25519 key for a Server to present.
func GenerateHostKey(rnd random.Service) (ssh.Signer, error) {
	_, private, err := ed25519.GenerateKey(rnd.Reader())
	if err != nil {
		return nil, fmt.Errorf("generating a host key: %w", err)
	}

	hostKey, err := ssh.NewSignerFromKey(private)
	if err != nil {
		return nil, fmt.Errorf("generating a host key: %w", err)
	}

	return hostKey, nil
}

// Serve accepts connections from listener until ctx ends or accepting fails.
// Then it closes listener and every connection, and waits for them to end.
// It returns nil if ctx ended.
func (s *Server) Serve(ctx context.Context, listener net.Listener) error {
	var connections sync.WaitGroup
	defer connections.Wait()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stopClosing := context.AfterFunc(ctx, func() {
		_ = listener.Close()
	})
	defer stopClosing()

	for {
		connection, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			return fmt.Errorf("accepting a connection: %w", err)
		}

		connections.Go(func() {
			s.handle(ctx, connection)
		})
	}
}

func (s *Server) handle(ctx context.Context, connection net.Conn) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// The SSH library takes no context, so closing the connection is what
	// ends a session when the server stops or the host goes away.
	context.AfterFunc(ctx, func() {
		_ = connection.Close()
	})

	logger := slog.With("client_address", connection.RemoteAddr().String())

	client, channels, admissionTicket, err := s.admit(ctx, connection)
	if err != nil {
		logger.Warn("refused a connection", "error", err)

		return
	}

	session, err := s.sessions.PrepareSession(ctx, admissionTicket)
	if err != nil {
		logger.Warn("the authority prepared no session", "error", err)

		return
	}

	logger = logger.With("session_id", string(session.SessionID))

	host, err := target.Connect(ctx, s.sessions, session)
	if err != nil {
		logger.Warn("no SSH connection to the host", "error", err)

		return
	}

	go func() {
		_ = host.Wait()

		cancel()
	}()

	logger.Info("session opened", "user_name", session.UserName)

	var relays sync.WaitGroup

	for newChannel := range channels {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(ssh.Prohibited, "only session channels are allowed")

			continue
		}

		relays.Go(func() {
			relayChannel(logger, newChannel, host)
		})
	}

	_ = host.Close()
	_ = client.Close()

	relays.Wait()

	logger.Info("session closed")
}

// admit reads the preamble, checks its ticket and runs the SSH handshake
// with the client. It only succeeds for a client that holds the key that the
// ticket names.
func (s *Server) admit(ctx context.Context, connection net.Conn) (*ssh.ServerConn, <-chan ssh.NewChannel, secret.String, error) {
	ctx, cancel := context.WithTimeoutCause(ctx, handshakeTimeout, errors.New("the client took too long to open its SSH connection"))
	defer cancel()

	stopClosing := context.AfterFunc(ctx, func() {
		_ = connection.Close()
	})

	reader := newPreambleReader(connection)

	admissionTicket, err := readPreamble(reader)
	if err != nil {
		return nil, nil, secret.String{}, err
	}

	claims, err := s.tickets.Verify(admissionTicket)
	if err != nil {
		return nil, nil, secret.String{}, fmt.Errorf("checking the ticket: %w", err)
	}

	config := &ssh.ServerConfig{
		PublicKeyCallback: func(_ ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if ssh.FingerprintSHA256(key) != claims.ClientKeyFingerprint {
				return nil, errors.New("the client's key isn't the key that the ticket names")
			}

			return &ssh.Permissions{}, nil
		},
	}
	config.AddHostKey(s.hostKey)

	client, channels, requests, err := ssh.NewServerConn(&preambleConn{Conn: connection, reader: reader}, config)
	if err != nil {
		return nil, nil, secret.String{}, fmt.Errorf("opening the client's SSH connection: %w", err)
	}

	if !stopClosing() {
		_ = client.Close()

		return nil, nil, secret.String{}, context.Cause(ctx)
	}

	go ssh.DiscardRequests(requests)

	return client, channels, admissionTicket, nil
}

// relayChannel opens a session on the host for the client's channel and
// relays between the two until one side ends.
func relayChannel(logger *slog.Logger, newChannel ssh.NewChannel, host *ssh.Client) {
	hostChannel, hostRequests, err := host.OpenChannel("session", nil)
	if err != nil {
		logger.Warn("the host opened no session", "error", err)

		_ = newChannel.Reject(ssh.ConnectionFailed, "the host opened no session")

		return
	}

	clientChannel, clientRequests, err := newChannel.Accept()
	if err != nil {
		_ = hostChannel.Close()

		return
	}

	relay(clientChannel, clientRequests, hostChannel, hostRequests)
}
