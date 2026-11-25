package proxy_test

import (
	"context"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/tidegate-dev/tidegate/internal/secret"
	"github.com/tidegate-dev/tidegate/ssh-proxy/internal/authority"
	"github.com/tidegate-dev/tidegate/ssh-proxy/internal/proxy"
	"github.com/tidegate-dev/tidegate/ssh-proxy/internal/sshtest"
	"github.com/tidegate-dev/tidegate/ssh-proxy/internal/ticket"
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

func TestClientShellGetsBackWhatItSends(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t)
	session := setup.session(t)

	input, err := session.StdinPipe()
	require.NoError(t, err)

	output, err := session.StdoutPipe()
	require.NoError(t, err)

	require.NoError(t, session.RequestPty("xterm", 24, 80, ssh.TerminalModes{}))
	require.NoError(t, session.Shell())

	// act
	_, err = io.WriteString(input, "ping\n")
	require.NoError(t, err)
	require.NoError(t, input.Close())

	// assert
	echoed, err := io.ReadAll(output)
	require.NoError(t, err)
	assert.Equal(t, "ping\n", string(echoed))
	require.NoError(t, session.Wait())
	assert.Equal(t, []string{"pty-req", "shell"}, setup.host.Requests())
}

func TestLocaleVariableReachesTheHost(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t)
	session := setup.session(t)

	// act
	err := session.Setenv("LC_ALL", "C")

	// assert
	require.NoError(t, err)
	assert.Equal(t, []string{"env LC_ALL"}, setup.host.Requests())
}

func TestRequestThatIsNotAllowedFailsAndNeverReachesTheHost(t *testing.T) {
	t.Parallel()

	variable := func(name string, value string) []byte {
		return ssh.Marshal(struct{ Name, Value string }{name, value})
	}

	tests := map[string]struct {
		requestType string
		payload     []byte
	}{
		"agent forwarding":            {requestType: "auth-agent-req@openssh.com"},
		"X11 forwarding":              {requestType: "x11-req"},
		"subsystem":                   {requestType: "subsystem", payload: ssh.Marshal(struct{ Name string }{"sftp"})},
		"signal":                      {requestType: "signal", payload: ssh.Marshal(struct{ Name string }{"KILL"})},
		"variable outside the locale": {requestType: "env", payload: variable("LD_PRELOAD", "/tmp/x.so")},
		"locale variable too long":    {requestType: "env", payload: variable("LANG", string(make([]byte, 1025)))},
		"variable that can't be read": {requestType: "env", payload: []byte{0, 0}},
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
	_, err := setup.connect(t, setup.ticket, anotherKey)

	// assert
	require.Error(t, err)
	assert.Zero(t, setup.sessions.prepared())
}

func TestTicketThatTheVerifierRefusesGetsNoSSHConnection(t *testing.T) {
	t.Parallel()

	// arrange
	verifier := &refusingVerifier{}
	setup := startWith(t, verifier)
	verifier.claims = ticket.Claims{ClientKeyFingerprint: setup.ticket}

	// act
	_, err := setup.connect(t, setup.ticket, setup.clientKey)

	// assert
	require.Error(t, err)
	assert.Zero(t, setup.sessions.prepared())
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
	assert.Zero(t, setup.sessions.prepared())
}

func TestHostWithAnotherKeyThanExpectedEndsTheConnectionWithoutAPassword(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t)
	setup.sessions.session.HostKey = sshtest.AuthorizedKey(sshtest.NewSigner(t).PublicKey())

	// act
	client, err := setup.connect(t, setup.ticket, setup.clientKey)
	require.NoError(t, err)

	// assert
	waitUntilClosed(t, client)
	assert.Zero(t, setup.sessions.issued())
}

func TestTicketThatTheAuthorityRefusesEndsTheConnection(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t)
	setup.sessions.prepareErr = assert.AnError

	// act
	client, err := setup.connect(t, setup.ticket, setup.clientKey)
	require.NoError(t, err)

	// assert
	waitUntilClosed(t, client)
	assert.Zero(t, setup.sessions.issued())
}

func TestServeClosesItsConnectionsWhenTheContextEnds(t *testing.T) {
	t.Parallel()

	// arrange
	setup := start(t)
	session := setup.session(t)
	require.NoError(t, session.Shell())

	// act
	setup.stop()

	// assert
	waitUntilClosed(t, setup.client(t))
	require.NoError(t, <-setup.served)
}

type setup struct {
	address   string
	proxyKey  ssh.PublicKey
	clientKey ssh.Signer
	// ticket is the fingerprint of clientKey, which ticket.Unsigned accepts.
	ticket   string
	host     *sshtest.Host
	sessions *sessions
	stop     context.CancelFunc
	served   chan error

	connectOnce sync.Once
	connected   *ssh.Client
}

// refusingVerifier refuses every ticket, and returns claims that would let
// the client in if the server used them anyway.
type refusingVerifier struct {
	claims ticket.Claims
}

func (v *refusingVerifier) Verify(secret.String) (ticket.Claims, error) {
	return v.claims, ticket.ErrInvalid
}

// start runs an SSH host and an ssh-proxy in front of it until the test
// ends. The ssh-proxy takes the fingerprint of the client's key as the
// ticket.
func start(t *testing.T) *setup {
	t.Helper()

	return startWith(t, ticket.Unsigned{})
}

func startWith(t *testing.T, tickets ticket.Verifier) *setup {
	t.Helper()

	host := sshtest.Start(t, userName, secret.New(password))
	proxyKey := sshtest.NewSigner(t)
	clientKey := sshtest.NewSigner(t)

	authorityCalls := &sessions{
		session: authority.PreparedSession{
			SessionID: "session-1",
			Host:      host.Host,
			Port:      host.Port,
			UserName:  userName,
			HostKey:   host.HostKey,
		},
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ctx, stop := context.WithCancel(context.Background())
	served := make(chan error, 1)

	go func() {
		served <- proxy.NewServer(proxyKey, tickets, authorityCalls).Serve(ctx, listener)
	}()

	t.Cleanup(stop)

	return &setup{
		address:   listener.Addr().String(),
		proxyKey:  proxyKey.PublicKey(),
		clientKey: clientKey,
		ticket:    ssh.FingerprintSHA256(clientKey.PublicKey()),
		host:      host,
		sessions:  authorityCalls,
		stop:      stop,
		served:    served,
	}
}

func (s *setup) clientConfig(key ssh.Signer) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User:            "ignored",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(key)},
		HostKeyCallback: ssh.FixedHostKey(s.proxyKey),
		Timeout:         waitTime,
	}
}

// connect sends the preamble with ticket and then opens an SSH connection
// with key.
func (s *setup) connect(t *testing.T, ticket string, key ssh.Signer) (*ssh.Client, error) {
	t.Helper()

	connection, err := net.Dial("tcp", s.address)
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })

	_, err = io.WriteString(connection, "tidegate-session-v1 "+ticket+"\n")
	require.NoError(t, err)

	clientConn, channels, requests, err := ssh.NewClientConn(connection, s.address, s.clientConfig(key))
	if err != nil {
		return nil, err
	}

	return ssh.NewClient(clientConn, channels, requests), nil
}

// client returns the test's one connection with the right ticket and key.
func (s *setup) client(t *testing.T) *ssh.Client {
	t.Helper()

	s.connectOnce.Do(func() {
		client, err := s.connect(t, s.ticket, s.clientKey)
		require.NoError(t, err)

		s.connected = client
	})

	require.NotNil(t, s.connected)

	return s.connected
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

// sessions stands in for the authority and counts its calls.
type sessions struct {
	session    authority.PreparedSession
	prepareErr error

	mu           sync.Mutex
	prepareCalls int
	issueCalls   int
}

func (s *sessions) PrepareSession(_ context.Context, _ secret.String) (authority.PreparedSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.prepareCalls++

	return s.session, s.prepareErr
}

func (s *sessions) IssueSessionCredential(_ context.Context, _ authority.SessionID, _ string) (secret.String, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.issueCalls++

	return secret.New(password), nil
}

func (s *sessions) prepared() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.prepareCalls
}

func (s *sessions) issued() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.issueCalls
}
