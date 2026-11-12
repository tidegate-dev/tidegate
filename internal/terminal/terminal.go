// Package terminal reads secrets from a terminal without showing them.
package terminal

import (
	"context"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// Terminal is a terminal that a person types at.
type Terminal struct {
	in       *os.File
	fallback io.Writer
}

// New returns the terminal that in reads from. It returns false if in isn't
// a terminal. The terminal writes to fallback only if it can't be opened for
// writing.
func New(in io.Reader, fallback io.Writer) (*Terminal, bool) {
	file, ok := in.(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) { //nolint:gosec // A file descriptor fits in an int.
		return nil, false
	}

	return &Terminal{in: file, fallback: fallback}, true
}

// ReadSecret writes prompt to the terminal and reads a line without echoing
// it. It stops waiting when ctx ends.
func (t *Terminal) ReadSecret(ctx context.Context, prompt string) (string, error) {
	out, closeOutput := t.output()
	defer closeOutput()

	fd := int(t.in.Fd()) //nolint:gosec // A file descriptor fits in an int.

	state, err := term.GetState(fd)
	if err != nil {
		return "", fmt.Errorf("reading the terminal's state: %w", err)
	}

	defer func() {
		_ = term.Restore(fd, state)
	}()

	_, err = fmt.Fprint(out, prompt)
	if err != nil {
		return "", err
	}

	// The Enter isn't echoed either.
	defer func() {
		_, _ = fmt.Fprintln(out)
	}()

	type readResult struct {
		secret []byte
		err    error
	}

	readResultChan := make(chan readResult, 1)

	// ReadPassword can't be canceled, so on ctx's end it's left running.
	go func() {
		secret, err := term.ReadPassword(fd)
		readResultChan <- readResult{secret, err}
	}()

	select {
	case result := <-readResultChan:
		if result.err != nil {
			return "", fmt.Errorf("reading from the terminal: %w", result.err)
		}

		return string(result.secret), nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// Like sudo and ssh, because stderr may go into a log file while the user
// types at the terminal.
func (t *Terminal) output() (io.Writer, func()) {
	tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		return t.fallback, func() {}
	}

	return tty, func() {
		_ = tty.Close()
	}
}
