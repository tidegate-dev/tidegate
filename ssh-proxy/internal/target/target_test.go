package target_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/tidegate-dev/tidegate/internal/secret"
	"github.com/tidegate-dev/tidegate/ssh-proxy/internal/authority"
	"github.com/tidegate-dev/tidegate/ssh-proxy/internal/sshtest"
	"github.com/tidegate-dev/tidegate/ssh-proxy/internal/target"
)

const (
	userName = "deploy"
	password = "correct horse"
)

func TestConnectLogsInToAHostThatPresentsTheExpectedKey(t *testing.T) {
	t.Parallel()

	// arrange
	host := sshtest.Start(t, userName, secret.New(password))
	credentials := &credentials{password: password}

	// act
	client, err := target.Connect(context.Background(), credentials, prepared(host))

	// assert
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	output, err := run(client, "hello")
	require.NoError(t, err)
	assert.Equal(t, "hello\n", output)
}

func TestConnectNamesTheSessionAndTheObservedHostKeyWhenItAsksForThePassword(t *testing.T) {
	t.Parallel()

	// arrange
	host := sshtest.Start(t, userName, secret.New(password))
	credentials := &credentials{password: password}

	// act
	client, err := target.Connect(context.Background(), credentials, prepared(host))

	// assert
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	assert.Equal(t, []issue{{sessionID: "session-1", observedHostKey: host.HostKey}}, credentials.issues())
}

func TestConnectRefusesAHostWithAnotherKeyBeforeItAsksForThePassword(t *testing.T) {
	t.Parallel()

	// arrange
	host := sshtest.Start(t, userName, secret.New(password))
	credentials := &credentials{password: password}

	session := prepared(host)
	session.HostKey = sshtest.AuthorizedKey(sshtest.NewSigner(t).PublicKey())

	// act
	_, err := target.Connect(context.Background(), credentials, session)

	// assert
	require.ErrorIs(t, err, target.ErrHostKeyMismatch)
	assert.Empty(t, credentials.issues())
}

func TestConnectFailsWhenTheAuthorityIssuesNoPassword(t *testing.T) {
	t.Parallel()

	// arrange
	host := sshtest.Start(t, userName, secret.New(password))
	credentials := &credentials{err: errors.New("the session is no longer authorized")}

	// act
	_, err := target.Connect(context.Background(), credentials, prepared(host))

	// assert
	require.Error(t, err)
}

func TestConnectFailsWhenTheHostRefusesThePassword(t *testing.T) {
	t.Parallel()

	// arrange
	host := sshtest.Start(t, userName, secret.New(password))
	credentials := &credentials{password: "another password"}

	// act
	_, err := target.Connect(context.Background(), credentials, prepared(host))

	// assert
	require.Error(t, err)
}

func TestConnectFailsForAnExpectedHostKeyThatIsNoPublicKey(t *testing.T) {
	t.Parallel()

	// arrange
	host := sshtest.Start(t, userName, secret.New(password))
	credentials := &credentials{password: password}

	session := prepared(host)
	session.HostKey = "ssh-ed25519 not-a-key"

	// act
	_, err := target.Connect(context.Background(), credentials, session)

	// assert
	require.Error(t, err)
	assert.Empty(t, credentials.issues())
}

func TestConnectStopsWhenTheContextEnds(t *testing.T) {
	t.Parallel()

	// arrange
	host := sshtest.Start(t, userName, secret.New(password))
	credentials := &credentials{password: password}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// act
	_, err := target.Connect(ctx, credentials, prepared(host))

	// assert
	require.Error(t, err)
	assert.Empty(t, credentials.issues())
}

func prepared(host *sshtest.Host) authority.PreparedSession {
	return authority.PreparedSession{
		SessionID: "session-1",
		Host:      host.Host,
		Port:      host.Port,
		UserName:  userName,
		HostKey:   host.HostKey,
	}
}

func run(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}

	defer func() {
		_ = session.Close()
	}()

	output, err := session.Output(command)

	return string(output), err
}

type issue struct {
	sessionID       authority.SessionID
	observedHostKey string
}

// credentials stands in for the authority and keeps what it was asked.
type credentials struct {
	password string
	err      error

	mu     sync.Mutex
	issued []issue
}

func (c *credentials) IssueSessionCredential(_ context.Context, sessionID authority.SessionID, observedHostKey string) (secret.String, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.issued = append(c.issued, issue{sessionID: sessionID, observedHostKey: observedHostKey})

	if c.err != nil {
		return secret.String{}, c.err
	}

	return secret.New(c.password), nil
}

func (c *credentials) issues() []issue {
	c.mu.Lock()
	defer c.mu.Unlock()

	return append([]issue(nil), c.issued...)
}
