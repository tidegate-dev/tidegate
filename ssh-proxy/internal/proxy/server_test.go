package proxy_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/phemera-dev/phemera/internal/secret"
	"github.com/phemera-dev/phemera/ssh-proxy/internal/authority"
	"github.com/phemera-dev/phemera/ssh-proxy/internal/proxy"
	"github.com/phemera-dev/phemera/ssh-proxy/internal/sshtest"
	"github.com/phemera-dev/phemera/ssh-proxy/internal/ticket"
)

const (
	userName = "deploy"
	password = "correct horse"
	waitTime = 5 * time.Second
)

func TestClientRunsACommandOnTheHost(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t)
	session := setup.session(t)

	// act
	output, err := session.Output("hello")

	// assert
	require.NoError(t, err)
	assert.Equal(t, "hello\n", string(output))
}

func TestClientGetsTheStandardErrorOfACommand(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t)
	session := setup.session(t)

	var standardError bytes.Buffer

	session.Stderr = &standardError

	// act
	err := session.Run("stderr oops")

	// assert
	require.NoError(t, err)
	assert.Equal(t, "oops\n", standardError.String())
}

func TestClientGetsTheExitStatusOfACommand(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t)
	session := setup.session(t)

	// act
	err := session.Run("false")

	// assert
	var exit *ssh.ExitError

	require.ErrorAs(t, err, &exit)
	assert.Equal(t, 1, exit.ExitStatus())
}

func TestClientGetsTheSignalThatEndedACommand(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t)
	session := setup.session(t)

	// act
	err := session.Run("signal")

	// assert
	var exit *ssh.ExitError

	require.ErrorAs(t, err, &exit)
	assert.Equal(t, "TERM", exit.Signal())
}

func TestClientShellGetsBackWhatItSends(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t)
	session := setup.session(t)

	input, err := session.StdinPipe()
	require.NoError(t, err)

	output, err := session.StdoutPipe()
	require.NoError(t, err)

	require.NoError(t, session.Shell())

	// act
	_, err = io.WriteString(input, "ping\n")
	require.NoError(t, err)
	require.NoError(t, input.Close())

	// assert
	echoed, err := io.ReadAll(output)
	require.NoError(t, err)
	assert.Equal(t, "ping\n", string(echoed))
}

func TestRequestThatIsAllowedReachesTheHost(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		requestType string
		payload     []byte
		reached     string
	}{
		"terminal": {
			requestType: "pty-req",
			payload: ssh.Marshal(struct {
				Term                         string
				Columns, Rows, Width, Height uint32
				Modes                        string
			}{Term: "xterm", Columns: 80, Rows: 24}),
			reached: "pty-req",
		},
		"terminal size": {
			requestType: "window-change",
			payload:     ssh.Marshal(struct{ Columns, Rows, Width, Height uint32 }{Columns: 100, Rows: 40}),
			reached:     "window-change",
		},
		"LANG":            {requestType: "env", payload: variable("LANG", "C"), reached: "env LANG"},
		"locale category": {requestType: "env", payload: variable("LC_ALL", "C"), reached: "env LC_ALL"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// arrange
			setup := start(t)
			session := setup.session(t)

			// act
			ok, err := session.SendRequest(test.requestType, true, test.payload)

			// assert
			require.NoError(t, err)
			assert.True(t, ok)
			assert.Equal(t, []string{test.reached}, setup.host.Requests())
		})
	}
}

func TestRequestThatIsNotAllowedFailsAndNeverReachesTheHost(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		requestType string
		payload     []byte
	}{
		"agent forwarding":                    {requestType: "auth-agent-req@openssh.com"},
		"X11 forwarding":                      {requestType: "x11-req"},
		"subsystem":                           {requestType: "subsystem", payload: ssh.Marshal(struct{ Name string }{"sftp"})},
		"signal":                              {requestType: "signal", payload: ssh.Marshal(struct{ Name string }{"KILL"})},
		"variable outside the locale":         {requestType: "env", payload: variable("LD_PRELOAD", "/tmp/x.so")},
		"locale variable too long":            {requestType: "env", payload: variable("LANG", strings.Repeat("a", 1025))},
		"locale variable without a category":  {requestType: "env", payload: variable("LC_", "C")},
		"locale variable with a line break":   {requestType: "env", payload: variable("LC_X=1\nLD_PRELOAD", "C")},
		"locale variable with a long name":    {requestType: "env", payload: variable("LC_"+strings.Repeat("A", 62), "C")},
		"locale variable in lower case":       {requestType: "env", payload: variable("LC_all", "C")},
		"variable that only starts with LANG": {requestType: "env", payload: variable("LANGUAGE", "C")},
		"variable that can't be read":         {requestType: "env", payload: []byte{0, 0}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// arrange
			setup := start(t)
			session := setup.session(t)

			// act
			ok, err := session.SendRequest(test.requestType, true, test.payload)

			// assert
			require.NoError(t, err)
			assert.False(t, ok)
			assert.Empty(t, setup.host.Requests())
		})
	}
}

func TestRequestFromTheHostThatIsNotAllowedFailsAndNeverReachesTheClient(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t)

	channel, requests, err := setup.client(t).OpenChannel("session", nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = channel.Close() })

	received := make(chan []string, 1)

	go func() {
		var requestTypes []string

		for request := range requests {
			requestTypes = append(requestTypes, request.Type)

			if request.WantReply {
				_ = request.Reply(true, nil)
			}
		}

		received <- requestTypes
	}()

	// act
	ok, err := channel.SendRequest("exec", true, ssh.Marshal(struct{ Command string }{"request keepalive@openssh.com"}))
	require.NoError(t, err)
	require.True(t, ok)

	// assert
	output, err := io.ReadAll(channel)
	require.NoError(t, err)
	assert.Equal(t, "false\n", string(output))
	assert.Equal(t, []string{"exit-status"}, <-received)
}

func TestForwardingAPortThroughTheHostIsRefused(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t)
	client := setup.client(t)

	// act
	_, err := client.Dial("tcp", "127.0.0.1:22")

	// assert
	var refused *ssh.OpenChannelError

	require.ErrorAs(t, err, &refused)
	assert.Equal(t, ssh.Prohibited, refused.Reason)
}

func TestListeningOnAPortOfTheHostIsRefused(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t)
	client := setup.client(t)

	// act
	_, err := client.Listen("tcp", "127.0.0.1:0")

	// assert
	require.Error(t, err)
}

func TestClientWithAnotherKeyThanTheTicketNamesIsRefused(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t)
	anotherKey := sshtest.NewSigner(t)

	// act
	_, err := setup.connect(t, anotherKey)

	// assert
	require.Error(t, err)
	assert.Zero(t, setup.sessions.prepared.Load())
}

func TestClientThatSignsWithSHA1IsRefused(t *testing.T) {
	t.Parallel()

	// arrange
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	signer, err := ssh.NewSignerFromKey(rsaKey)
	require.NoError(t, err)

	withSHA1, err := ssh.NewSignerWithAlgorithms(signer.(ssh.AlgorithmSigner), []string{ssh.KeyAlgoRSA})
	require.NoError(t, err)

	setup := start(t, func(s *setup) {
		s.verifier.claims.ClientKeyFingerprint = ssh.FingerprintSHA256(signer.PublicKey())
	})

	// act
	_, err = setup.connect(t, withSHA1)

	// assert
	require.Error(t, err)
	assert.Zero(t, setup.sessions.prepared.Load())
}

func TestTicketThatTheVerifierRefusesGetsNoSSHConnection(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t, func(s *setup) { s.verifier.err = assert.AnError })

	// act
	_, err := setup.connect(t, setup.clientKey)

	// assert
	require.Error(t, err)
	assert.Zero(t, setup.sessions.prepared.Load())
}

func TestConnectionWithoutAPreambleIsClosed(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t)

	connection, err := net.Dial("tcp", setup.address)
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })

	// act
	_, _, _, err = ssh.NewClientConn(connection, setup.address, setup.clientConfig(setup.clientKey))

	// assert
	require.Error(t, err)
	assert.Zero(t, setup.sessions.prepared.Load())
}

func TestHostWithAnotherKeyThanExpectedGetsNoPassword(t *testing.T) {
	t.Parallel()

	// arrange
	anotherKey := sshtest.AuthorizedKey(sshtest.NewSigner(t).PublicKey())
	setup := start(t, func(s *setup) { s.sessions.session.HostKey = anotherKey })
	client := setup.client(t)

	// act
	_, err := client.NewSession()

	// assert
	require.Error(t, err)
	waitUntilClosed(t, client)
	assert.Zero(t, setup.sessions.issued.Load())
}

func TestClientIsToldWhenTheAuthorityRefusesItsTicket(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t, func(s *setup) { s.sessions.prepareErr = assert.AnError })
	client := setup.client(t)

	// act
	_, err := client.NewSession()

	// assert
	var refused *ssh.OpenChannelError

	require.ErrorAs(t, err, &refused)
	assert.Equal(t, ssh.ConnectionFailed, refused.Reason)
}

func TestTicketThatTheAuthorityRefusesEndsTheConnection(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t, func(s *setup) { s.sessions.prepareErr = assert.AnError })
	client := setup.client(t)

	// act
	_, err := client.NewSession()

	// assert
	require.Error(t, err)
	waitUntilClosed(t, client)
}

func TestClientThatLeavesWhileItsSessionIsPreparedEndsThePreparation(t *testing.T) {
	t.Parallel()

	// arrange
	canceled := make(chan struct{})
	setup := start(t, func(s *setup) { s.sessions.canceled = canceled })
	client := setup.client(t)

	// act
	require.NoError(t, client.Close())

	// assert
	select {
	case <-canceled:
	case <-time.After(waitTime):
		t.Fatal("the authority is still preparing the session")
	}
}

func TestClientGetsTheEndOfTheOutputWhileTheCommandStillRuns(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t)
	session := setup.session(t)

	output, err := session.StdoutPipe()
	require.NoError(t, err)

	ended := make(chan error, 1)

	// act
	require.NoError(t, session.Start("eof"))

	go func() {
		_, err := io.ReadAll(output)
		ended <- err
	}()

	// assert
	select {
	case err := <-ended:
		require.NoError(t, err)
	case <-time.After(waitTime):
		t.Fatal("the client got no end of the output")
	}
}

func TestClientThatClosesItsChannelEndsTheSessionOnTheHost(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t)
	session := setup.session(t)
	require.NoError(t, session.Start("wait"))

	// act
	require.NoError(t, session.Close())

	// assert
	select {
	case <-setup.host.SessionEnded:
	case <-time.After(waitTime):
		t.Fatal("the session on the host is still open")
	}
}

func TestHostThatHangsUpEndsTheClientConnection(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t)
	client := setup.client(t)

	session, err := client.NewSession()
	require.NoError(t, err)

	// act
	_ = session.Start("hang up")

	// assert
	waitUntilClosed(t, client)
}

func TestServeAcceptsAgainAfterATemporaryFailure(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t, func(s *setup) { s.listener = &failingOnceListener{Listener: s.listener} })
	session := setup.session(t)

	// act
	output, err := session.Output("hello")

	// assert
	require.NoError(t, err)
	assert.Equal(t, "hello\n", string(output))
}

func TestServeClosesItsConnectionsWhenTheContextEnds(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t)
	client := setup.client(t)

	session, err := client.NewSession()
	require.NoError(t, err)
	require.NoError(t, session.Shell())

	// act
	setup.stop()

	// assert
	waitUntilClosed(t, client)
	require.NoError(t, setup.waitForServe(t))
}

func TestServeStopsWhileAHostLeavesARequestUnanswered(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t)
	session := setup.session(t)
	require.NoError(t, session.Start("stall"))

	terminalSize := ssh.Marshal(struct{ Columns, Rows, Width, Height uint32 }{Columns: 100, Rows: 40})

	go func() {
		_, _ = session.SendRequest("window-change", true, terminalSize)
	}()

	// The SSH library holds 16 requests of a channel before it stops reading the connection.
	for range 32 {
		_, err := session.SendRequest("window-change", false, terminalSize)
		require.NoError(t, err)
	}

	time.Sleep(200 * time.Millisecond)

	// act
	setup.stop()

	// assert
	select {
	case <-setup.served:
	case <-time.After(waitTime):
		t.Fatal("the ssh-proxy is still serving")
	}
}

func TestServeClosesTheListenerWhenAcceptingFailsForGood(t *testing.T) {
	t.Parallel()

	// arrange
	listener := &failingListener{closed: make(chan struct{})}
	server := proxy.NewServer(sshtest.NewSigner(t), &verifier{}, &sessions{})

	// act
	err := server.Serve(context.Background(), listener)

	// assert
	require.ErrorIs(t, err, assert.AnError)

	select {
	case <-listener.closed:
	default:
		t.Fatal("the listener is still open")
	}
}

func variable(name string, value string) []byte {
	return ssh.Marshal(struct{ Name, Value string }{name, value})
}

type setup struct {
	address   string
	listener  net.Listener
	proxyKey  ssh.PublicKey
	clientKey ssh.Signer
	host      *sshtest.Host
	verifier  *verifier
	sessions  *sessions
	stop      context.CancelFunc

	served   chan struct{}
	serveErr error
}

// start runs an SSH host and an ssh-proxy in front of it until the test
// ends. Every ticket is bound to clientKey. A configure function can change
// the listener and the stand-ins for the ticket check and the authority
// before the ssh-proxy starts.
func start(t *testing.T, configure ...func(*setup)) *setup {
	t.Helper()

	host := sshtest.Start(t, userName, secret.New(password))
	proxyKey := sshtest.NewSigner(t)
	clientKey := sshtest.NewSigner(t)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ctx, stop := context.WithCancel(context.Background())

	s := &setup{
		address:   listener.Addr().String(),
		listener:  listener,
		proxyKey:  proxyKey.PublicKey(),
		clientKey: clientKey,
		host:      host,
		verifier:  &verifier{claims: ticket.Claims{ClientKeyFingerprint: ssh.FingerprintSHA256(clientKey.PublicKey())}},
		sessions: &sessions{
			session: authority.PreparedSession{
				SessionID: "session-1",
				Host:      host.Address,
				Port:      host.Port,
				UserName:  userName,
				HostKey:   host.HostKey,
			},
		},
		stop:   stop,
		served: make(chan struct{}),
	}

	for _, change := range configure {
		change(s)
	}

	go func() {
		s.serveErr = proxy.NewServer(proxyKey, s.verifier, s.sessions).Serve(ctx, s.listener)

		close(s.served)
	}()

	t.Cleanup(func() {
		stop()

		_ = s.waitForServe(t)
	})

	return s
}

// waitForServe returns what Serve returned, and fails the test if Serve is
// still running after waitTime.
func (s *setup) waitForServe(t *testing.T) error {
	t.Helper()

	select {
	case <-s.served:
	case <-time.After(waitTime):
		t.Error("the ssh-proxy is still serving")
	}

	return s.serveErr
}

func (s *setup) clientConfig(key ssh.Signer) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User:            "ignored",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(key)},
		HostKeyCallback: ssh.FixedHostKey(s.proxyKey),
		Timeout:         waitTime,
	}
}

// connect sends a preamble and then opens an SSH connection with key. It
// closes the connection after twice waitTime, so a test that would wait for
// good fails.
func (s *setup) connect(t *testing.T, key ssh.Signer) (*ssh.Client, error) {
	t.Helper()

	connection, err := net.Dial("tcp", s.address)
	require.NoError(t, err)

	closeLate := time.AfterFunc(2*waitTime, func() { _ = connection.Close() })

	t.Cleanup(func() {
		closeLate.Stop()

		_ = connection.Close()
	})

	_, err = io.WriteString(connection, "phemera-session-v1 the-ticket\n")
	require.NoError(t, err)

	clientConn, channels, requests, err := ssh.NewClientConn(connection, s.address, s.clientConfig(key))
	if err != nil {
		return nil, err
	}

	return ssh.NewClient(clientConn, channels, requests), nil
}

// client opens an SSH connection with the key that the ticket is bound to.
func (s *setup) client(t *testing.T) *ssh.Client {
	t.Helper()

	client, err := s.connect(t, s.clientKey)
	require.NoError(t, err)

	return client
}

func (s *setup) session(t *testing.T) *ssh.Session {
	t.Helper()

	session, err := s.client(t).NewSession()
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	return session
}

func waitUntilClosed(t *testing.T, client *ssh.Client) {
	t.Helper()

	closed := make(chan struct{})

	go func() {
		_ = client.Wait()

		close(closed)
	}()

	select {
	case <-closed:
	case <-time.After(waitTime):
		t.Fatal("the connection is still open")
	}
}

// verifier stands in for the ticket check. It answers every ticket with the
// claims and the error that it holds.
type verifier struct {
	claims ticket.Claims
	err    error
}

func (v *verifier) Verify(secret.String) (ticket.Claims, error) {
	return v.claims, v.err
}

// sessions stands in for the authority and counts its calls.
type sessions struct {
	session    authority.PreparedSession
	prepareErr error
	// canceled makes PrepareSession wait until its context ends and then
	// close the channel.
	canceled chan struct{}

	prepared atomic.Int32
	issued   atomic.Int32
}

func (s *sessions) PrepareSession(ctx context.Context, _ secret.String) (authority.PreparedSession, error) {
	s.prepared.Add(1)

	if s.canceled != nil {
		<-ctx.Done()
		close(s.canceled)

		return authority.PreparedSession{}, ctx.Err()
	}

	return s.session, s.prepareErr
}

func (s *sessions) IssueSessionCredential(context.Context, authority.SessionID, string) (secret.String, error) {
	s.issued.Add(1)

	return secret.New(password), nil
}

// failingListener fails every Accept for good and reports when it is closed.
type failingListener struct {
	closed    chan struct{}
	closeOnce sync.Once
}

func (l *failingListener) Accept() (net.Conn, error) {
	return nil, assert.AnError
}

func (l *failingListener) Close() error {
	l.closeOnce.Do(func() { close(l.closed) })

	return nil
}

func (l *failingListener) Addr() net.Addr {
	return &net.TCPAddr{}
}

// failingOnceListener fails its first Accept the way a system does that has
// too many open files.
type failingOnceListener struct {
	net.Listener

	failed atomic.Bool
}

func (l *failingOnceListener) Accept() (net.Conn, error) {
	if l.failed.CompareAndSwap(false, true) {
		return nil, temporaryError{}
	}

	return l.Listener.Accept()
}

type temporaryError struct{}

func (temporaryError) Error() string { return "too many open files" }

func (temporaryError) Temporary() bool { return true }
