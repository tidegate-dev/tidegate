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

	"github.com/phemera-dev/phemera/internal/secret"
	"github.com/phemera-dev/phemera/ssh-proxy/internal/authority"
	"github.com/phemera-dev/phemera/ssh-proxy/internal/target"
	"github.com/phemera-dev/phemera/ssh-proxy/internal/ticket"
)

const (
	handshakeTimeout = 30 * time.Second
	prepareTimeout   = 10 * time.Second
	refusalTimeout   = 5 * time.Second

	minAcceptRetryDelay = 5 * time.Millisecond
	maxAcceptRetryDelay = time.Second

	maxLoggedErrorBytes = 256
)

var errHandshakeTimeout = errors.New("the client took too long to open its SSH connection")

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

// Serve accepts connections from listener until ctx ends or accepting fails
// for good. Then it closes listener and every connection, and waits for them
// to end. It returns nil if ctx ended. After a temporary failure, like too
// many open files, it waits and accepts again.
func (s *Server) Serve(ctx context.Context, listener net.Listener) error {
	var connections sync.WaitGroup
	defer connections.Wait()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	defer func() {
		_ = listener.Close()
	}()

	// Accept takes no context.
	context.AfterFunc(ctx, func() {
		_ = listener.Close()
	})

	var retryDelay time.Duration

	for {
		connection, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			var temporary interface{ Temporary() bool }
			if !errors.As(err, &temporary) || !temporary.Temporary() {
				return fmt.Errorf("accepting a connection: %w", err)
			}

			retryDelay = min(max(2*retryDelay, minAcceptRetryDelay), maxAcceptRetryDelay)
			slog.Warn("accepting a connection failed, trying again", "error", err, "retry_after", retryDelay)

			select {
			case <-time.After(retryDelay):
			case <-ctx.Done():
				return nil
			}

			continue
		}

		retryDelay = 0

		connections.Go(func() {
			s.handle(ctx, connection)
		})
	}
}

func (s *Server) handle(ctx context.Context, connection net.Conn) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// The SSH library takes no context.
	context.AfterFunc(ctx, func() {
		_ = connection.Close()
	})

	logger := slog.With("client_address", connection.RemoteAddr().String())

	client, channels, admissionTicket, err := s.admit(ctx, connection)
	if err != nil {
		logger.Warn("refused a connection", "error", logText(err))

		return
	}

	// The SSH library's goroutines for a connection only end once its channels are read to the end.
	defer func() {
		_ = client.Close()

		for newChannel := range channels {
			_ = newChannel.Reject(ssh.ConnectionFailed, "the session has ended")
		}
	}()

	go func() {
		_ = client.Wait()

		cancel()
	}()

	host, session, err := s.openHost(ctx, admissionTicket)
	if err != nil {
		logger.Warn("opened no session", "error", logText(err))
		refuseSession(ctx, channels)

		return
	}

	go func() {
		_ = host.Wait()

		cancel()
	}()

	// While the host leaves a request unanswered, the SSH library can stop reading the client's connection.
	context.AfterFunc(ctx, func() {
		_ = host.Close()
	})

	logger = logger.With("session_id", string(session.SessionID))
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

	relays.Wait()

	logger.Info("session closed")
}

// admit reads the preamble, checks its ticket and runs the SSH handshake
// with the client. It only succeeds for a client that holds the key that the
// ticket names, and it gives up after handshakeTimeout.
func (s *Server) admit(ctx context.Context, connection net.Conn) (*ssh.ServerConn, <-chan ssh.NewChannel, secret.String, error) {
	ctx, cancel := context.WithTimeoutCause(ctx, handshakeTimeout, errHandshakeTimeout)
	defer cancel()

	stopClosing := context.AfterFunc(ctx, func() {
		_ = connection.Close()
	})

	client, channels, admissionTicket, err := s.handshake(connection)

	if !stopClosing() {
		if client != nil {
			_ = client.Close()
		}

		return nil, nil, secret.String{}, context.Cause(ctx)
	}

	return client, channels, admissionTicket, err
}

func (s *Server) handshake(connection net.Conn) (*ssh.ServerConn, <-chan ssh.NewChannel, secret.String, error) {
	admissionTicket, reader, err := readPreamble(connection)
	if err != nil {
		return nil, nil, secret.String{}, err
	}

	claims, err := s.tickets.Verify(admissionTicket)
	if err != nil {
		return nil, nil, secret.String{}, fmt.Errorf("checking the ticket: %w", err)
	}

	// The library's defaults still accept SHA-1 in key exchanges and in RSA signatures, for old clients.
	algorithms := ssh.SupportedAlgorithms()

	config := &ssh.ServerConfig{
		Config:                  ssh.Config{KeyExchanges: algorithms.KeyExchanges},
		PublicKeyAuthAlgorithms: algorithms.PublicKeyAuths,
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

	go ssh.DiscardRequests(requests)

	return client, channels, admissionTicket, nil
}

// openHost asks the authority for the session of the ticket and logs in to
// the session's host.
func (s *Server) openHost(ctx context.Context, admissionTicket secret.String) (*ssh.Client, authority.PreparedSession, error) {
	prepareCtx, cancel := context.WithTimeout(ctx, prepareTimeout)
	defer cancel()

	session, err := s.sessions.PrepareSession(prepareCtx, admissionTicket)
	if err != nil {
		return nil, authority.PreparedSession{}, fmt.Errorf("preparing the session: %w", err)
	}

	host, err := target.Connect(ctx, target.Host{
		Address:  session.Host,
		Port:     session.Port,
		UserName: session.UserName,
		HostKey:  session.HostKey,
	}, func(ctx context.Context, observedHostKey string) (secret.String, error) {
		return s.sessions.IssueSessionCredential(ctx, session.SessionID, observedHostKey)
	})
	if err != nil {
		return nil, authority.PreparedSession{}, fmt.Errorf("session %s: %w", session.SessionID, err)
	}

	return host, session, nil
}

// refuseSession answers the client's next channel with a failure, so the
// client learns that no session opens. It gives up after refusalTimeout.
func refuseSession(ctx context.Context, channels <-chan ssh.NewChannel) {
	select {
	case newChannel, open := <-channels:
		if open {
			_ = newChannel.Reject(ssh.ConnectionFailed, "phemera opened no session")
		}
	case <-time.After(refusalTimeout):
	case <-ctx.Done():
	}
}

// relayChannel opens a session on the host for the client's channel and
// relays between the two until one side ends.
func relayChannel(logger *slog.Logger, newChannel ssh.NewChannel, host *ssh.Client) {
	hostChannel, hostRequests, err := host.OpenChannel("session", nil)
	if err != nil {
		logger.Warn("the host opened no session", "error", logText(err))

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

// logText returns the text of err, cut to maxLoggedErrorBytes. The SSH
// library puts what the peer sent into its errors.
func logText(err error) string {
	text := err.Error()
	if len(text) > maxLoggedErrorBytes {
		return text[:maxLoggedErrorBytes] + "..."
	}

	return text
}
