package sshsession_test

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/binary"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/The127/go-random"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/phemera-dev/phemera/cli/internal/sshsession"
)

// proxy stands in for the ssh-proxy. It reads the line before the SSH
// connection and serves a shell that writes back what it reads.
type proxy struct {
	address string
	// exitStatus is the status that a shell ends with.
	exitStatus uint32
	// refuseSessions makes the proxy refuse every session.
	refuseSessions bool

	// preamble gets the line before the SSH connection, without its line
	// break.
	preamble chan string
	// keyFingerprint gets the SHA256 fingerprint of the key that the client
	// logged in with.
	keyFingerprint chan string
}

func startProxy(t *testing.T) *proxy {
	t.Helper()

	_, private, err := ed25519.GenerateKey(random.NewCryptoRandom().Reader())
	require.NoError(t, err)

	hostKey, err := ssh.NewSignerFromKey(private)
	require.NoError(t, err)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	served := &proxy{
		address:        listener.Addr().String(),
		preamble:       make(chan string, 1),
		keyFingerprint: make(chan string, 1),
	}

	done := make(chan struct{})

	go func() {
		defer close(done)

		connection, err := listener.Accept()
		if err != nil {
			return
		}

		defer func() {
			_ = connection.Close()
		}()

		served.serve(connection, hostKey)
	}()

	t.Cleanup(func() {
		_ = listener.Close()

		<-done
	})

	return served
}

func (p *proxy) serve(connection net.Conn, hostKey ssh.Signer) {
	reader := bufio.NewReader(connection)

	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	p.preamble <- strings.TrimSuffix(line, "\n")

	config := &ssh.ServerConfig{
		PublicKeyCallback: func(_ ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			return &ssh.Permissions{Extensions: map[string]string{"fingerprint": ssh.FingerprintSHA256(key)}}, nil
		},
	}
	config.AddHostKey(hostKey)

	sshConnection, channels, requests, err := ssh.NewServerConn(&bufferedConn{Conn: connection, reader: reader}, config)
	if err != nil {
		return
	}

	p.keyFingerprint <- sshConnection.Permissions.Extensions["fingerprint"]

	go ssh.DiscardRequests(requests)

	for newChannel := range channels {
		if p.refuseSessions {
			_ = newChannel.Reject(ssh.Prohibited, "no sessions")

			continue
		}

		channel, channelRequests, err := newChannel.Accept()
		if err != nil {
			return
		}

		for request := range channelRequests {
			_ = request.Reply(request.Type == "shell", nil)

			if request.Type == "shell" {
				break
			}
		}

		_, _ = io.Copy(channel, channel)
		_, _ = channel.SendRequest("exit-status", false, binary.BigEndian.AppendUint32(nil, p.exitStatus))
		_ = channel.Close()
	}
}

// bufferedConn reads through the reader that read the preamble.
type bufferedConn struct {
	net.Conn

	reader *bufio.Reader
}

func (c *bufferedConn) Read(buffer []byte) (int, error) {
	return c.reader.Read(buffer)
}

func run(t *testing.T, address string, in string) (string, error) {
	t.Helper()

	var out bytes.Buffer

	err := sshsession.Run(context.Background(), random.NewCryptoRandom(), address, sshsession.Streams{
		In:  strings.NewReader(in),
		Out: &out,
		Err: io.Discard,
	})

	return out.String(), err
}

func TestRunSendsTheFingerprintOfItsKeyAsTheTicket(t *testing.T) {
	served := startProxy(t)

	_, err := run(t, served.address, "")

	require.NoError(t, err)
	assert.Equal(t, "phemera-session-v1 "+<-served.keyFingerprint, <-served.preamble)
}

func TestRunRelaysTheInputToTheShellAndTheShellsOutputBack(t *testing.T) {
	served := startProxy(t)

	out, err := run(t, served.address, "whoami\n")

	require.NoError(t, err)
	assert.Equal(t, "whoami\n", out)
}

func TestRunReturnsTheExitStatusOfAShellThatEndsWithAFailure(t *testing.T) {
	served := startProxy(t)
	served.exitStatus = 3

	_, err := run(t, served.address, "")

	var exitErr *sshsession.ExitError

	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 3, exitErr.Status)
}

func TestRunFailsWhenTheProxyRefusesTheSession(t *testing.T) {
	served := startProxy(t)
	served.refuseSessions = true

	_, err := run(t, served.address, "")

	assert.ErrorContains(t, err, "opening the session")
}

func TestRunFailsWhenNothingListensAtTheAddress(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	address := listener.Addr().String()
	require.NoError(t, listener.Close())

	_, err = run(t, address, "")

	assert.ErrorContains(t, err, "connecting to the ssh-proxy")
}

func TestRunStopsWhenTheContextEnds(t *testing.T) {
	served := startProxy(t)

	ctx, cancel := context.WithCancel(context.Background())
	in, closeInput := io.Pipe()

	defer func() {
		_ = closeInput.Close()
	}()

	failed := make(chan error, 1)

	go func() {
		failed <- sshsession.Run(ctx, random.NewCryptoRandom(), served.address, sshsession.Streams{
			In:  in,
			Out: io.Discard,
			Err: io.Discard,
		})
	}()

	<-served.keyFingerprint
	cancel()

	assert.ErrorIs(t, <-failed, context.Canceled)
}
