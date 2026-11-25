package target_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/phemera-dev/phemera/internal/secret"
	"github.com/phemera-dev/phemera/ssh-proxy/internal/sshtest"
	"github.com/phemera-dev/phemera/ssh-proxy/internal/target"
)

const (
	userName = "deploy"
	password = "correct horse"
)

func TestConnectLogsInToAHostThatPresentsTheExpectedKey(t *testing.T) {
	t.Parallel()

	// arrange
	host := sshtest.Start(t, userName, secret.New(password))

	// act
	client, err := target.Connect(context.Background(), expected(host), issue(password))

	// assert
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	output, err := run(client, "hello")
	require.NoError(t, err)
	assert.Equal(t, "hello\n", output)
}

func TestConnectNamesTheObservedHostKeyWhenItAsksForThePassword(t *testing.T) {
	t.Parallel()

	// arrange
	host := sshtest.Start(t, userName, secret.New(password))

	var observedHostKeys []string

	issuePassword := func(_ context.Context, observedHostKey string) (secret.String, error) {
		observedHostKeys = append(observedHostKeys, observedHostKey)

		return secret.New(password), nil
	}

	// act
	client, err := target.Connect(context.Background(), expected(host), issuePassword)

	// assert
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	assert.Equal(t, []string{host.HostKey}, observedHostKeys)
}

func TestConnectRefusesAHostWithAnotherKeyBeforeItAsksForThePassword(t *testing.T) {
	t.Parallel()

	// arrange
	host := sshtest.Start(t, userName, secret.New(password))

	withAnotherKey := expected(host)
	withAnotherKey.HostKey = sshtest.AuthorizedKey(sshtest.NewSigner(t).PublicKey())

	// act
	_, err := target.Connect(context.Background(), withAnotherKey, neverIssue(t))

	// assert
	require.ErrorIs(t, err, target.ErrHostKeyMismatch)
}

func TestConnectFailsWhenNoPasswordIsIssued(t *testing.T) {
	t.Parallel()

	// arrange
	host := sshtest.Start(t, userName, secret.New(password))

	issuePassword := func(context.Context, string) (secret.String, error) {
		return secret.String{}, errors.New("the session is no longer authorized")
	}

	// act
	_, err := target.Connect(context.Background(), expected(host), issuePassword)

	// assert
	require.Error(t, err)
}

func TestConnectFailsWhenTheHostRefusesThePassword(t *testing.T) {
	t.Parallel()

	// arrange
	host := sshtest.Start(t, userName, secret.New(password))

	// act
	_, err := target.Connect(context.Background(), expected(host), issue("another password"))

	// assert
	require.Error(t, err)
}

func TestConnectFailsForAnExpectedHostKeyThatIsNoPublicKey(t *testing.T) {
	t.Parallel()

	// arrange
	host := sshtest.Start(t, userName, secret.New(password))

	withoutAKey := expected(host)
	withoutAKey.HostKey = "ssh-ed25519 not-a-key"

	// act
	_, err := target.Connect(context.Background(), withoutAKey, neverIssue(t))

	// assert
	require.Error(t, err)
}

func TestConnectStopsWhenTheContextEnds(t *testing.T) {
	t.Parallel()

	// arrange
	host := sshtest.Start(t, userName, secret.New(password))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// act
	_, err := target.Connect(ctx, expected(host), neverIssue(t))

	// assert
	require.Error(t, err)
}

func TestConnectGivesUpOnAHostThatNeverAnswersWhenTheContextEnds(t *testing.T) {
	t.Parallel()

	// arrange
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	address := listener.Addr().(*net.TCPAddr).AddrPort()
	silent := target.Host{
		Address:  address.Addr().String(),
		Port:     address.Port(),
		UserName: userName,
		HostKey:  sshtest.AuthorizedKey(sshtest.NewSigner(t).PublicKey()),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// act
	_, err = target.Connect(ctx, silent, neverIssue(t))

	// assert
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

// expected returns what a caller knows about host before it connects.
func expected(host *sshtest.Host) target.Host {
	return target.Host{
		Address:  host.Address,
		Port:     host.Port,
		UserName: userName,
		HostKey:  host.HostKey,
	}
}

func issue(password string) target.IssuePassword {
	return func(context.Context, string) (secret.String, error) {
		return secret.New(password), nil
	}
}

// neverIssue fails the test if Connect asks for the password.
func neverIssue(t *testing.T) target.IssuePassword {
	t.Helper()

	return func(context.Context, string) (secret.String, error) {
		t.Error("Connect asked for the password")

		return secret.String{}, errors.New("no password")
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
