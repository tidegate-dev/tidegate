// Package sshsession opens a shell on an SSH host through the phemera
// ssh-proxy.
package sshsession

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/The127/go-random"
	"golang.org/x/crypto/ssh"

	"github.com/phemera-dev/phemera/internal/terminal"
)

const (
	preambleVersion = "phemera-session-v1"

	// The ssh-proxy ignores the user name.
	userName = "session"

	dialTimeout = 10 * time.Second

	defaultTerminalType = "xterm"
)

// Streams are the input and the outputs that a session is relayed to.
type Streams struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

// ExitError means that the shell ended with a status other than 0.
type ExitError struct {
	// Status is the shell's exit status.
	Status int
}

func (err *ExitError) Error() string {
	return fmt.Sprintf("the shell ended with status %d", err.Status)
}

// Run opens a shell through the ssh-proxy at address and relays it to streams
// until the shell ends or ctx ends. When the shell ended with a status other
// than 0, it returns an *ExitError. If the input is a terminal, the shell gets
// a terminal of the same size.
//
// Run logs in with a key that it generates, and names the key's fingerprint
// as the ticket. It accepts every host key that the ssh-proxy presents.
func Run(ctx context.Context, rnd random.Service, address string, streams Streams) error {
	err := run(ctx, rnd, address, streams)
	if err != nil && ctx.Err() != nil {
		return ctx.Err()
	}

	return err
}

func run(ctx context.Context, rnd random.Service, address string, streams Streams) error {
	key, err := generateKey(rnd)
	if err != nil {
		return err
	}

	dialer := net.Dialer{Timeout: dialTimeout}

	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("connecting to the ssh-proxy: %w", err)
	}

	defer func() {
		_ = connection.Close()
	}()

	// The SSH handshake and the session take no context.
	stop := context.AfterFunc(ctx, func() {
		_ = connection.Close()
	})
	defer stop()

	_, err = fmt.Fprintf(connection, "%s %s\n", preambleVersion, ssh.FingerprintSHA256(key.PublicKey()))
	if err != nil {
		return fmt.Errorf("sending the ticket to the ssh-proxy: %w", err)
	}

	sshConnection, channels, requests, err := ssh.NewClientConn(connection, address, &ssh.ClientConfig{
		User: userName,
		Auth: []ssh.AuthMethod{ssh.PublicKeys(key)},
		// The ssh-proxy creates a new host key every time it starts.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // The prototype has no host key to compare with.
	})
	if err != nil {
		return fmt.Errorf("logging in to the ssh-proxy: %w", err)
	}

	client := ssh.NewClient(sshConnection, channels, requests)

	defer func() {
		_ = client.Close()
	}()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("opening the session: %w", err)
	}

	defer func() {
		_ = session.Close()
	}()

	return relay(ctx, session, streams)
}

func generateKey(rnd random.Service) (ssh.Signer, error) {
	_, private, err := ed25519.GenerateKey(rnd.Reader())
	if err != nil {
		return nil, fmt.Errorf("generating a key for the session: %w", err)
	}

	key, err := ssh.NewSignerFromKey(private)
	if err != nil {
		return nil, fmt.Errorf("generating a key for the session: %w", err)
	}

	return key, nil
}

func relay(ctx context.Context, session *ssh.Session, streams Streams) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	session.Stdout = streams.Out
	session.Stderr = streams.Err

	input, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("opening the shell's input: %w", err)
	}

	tty := terminal.New(streams.In, streams.Err)
	if tty.Interactive() {
		restore, err := useTerminal(ctx, tty, session)
		if err != nil {
			return err
		}

		defer restore()
	}

	err = session.Shell()
	if err != nil {
		return fmt.Errorf("starting the shell: %w", err)
	}

	// A read from a terminal can't be canceled, so on the shell's end it's left running.
	go func() {
		_, _ = io.Copy(input, streams.In)
		_ = input.Close()
	}()

	err = session.Wait()

	exitErr, exited := errors.AsType[*ssh.ExitError](err)
	if exited {
		return &ExitError{Status: exitErr.ExitStatus()}
	}

	if err != nil {
		return fmt.Errorf("relaying the session: %w", err)
	}

	return nil
}

// useTerminal asks for a terminal of tty's size for the shell and makes tty
// pass on every key. The returned function puts tty back the way it was.
func useTerminal(ctx context.Context, tty *terminal.Terminal, session *ssh.Session) (func(), error) {
	width, height, err := tty.Size()
	if err != nil {
		return nil, err
	}

	terminalType := os.Getenv("TERM")
	if terminalType == "" {
		terminalType = defaultTerminalType
	}

	err = session.RequestPty(terminalType, height, width, ssh.TerminalModes{})
	if err != nil {
		return nil, fmt.Errorf("asking for a terminal: %w", err)
	}

	restore, err := tty.Raw()
	if err != nil {
		return nil, err
	}

	followSize(ctx, tty, session)

	return restore, nil
}
