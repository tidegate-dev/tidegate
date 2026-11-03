// Package terminal reads secrets from a terminal without showing them.
package terminal

import (
	"context"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// File returns in as a file if it's a terminal.
func File(in io.Reader) (*os.File, bool) {
	file, ok := in.(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) { //nolint:gosec // A file descriptor fits in an int.
		return nil, false
	}

	return file, true
}

// ReadSecret writes prompt to the terminal and reads a line from in without
// echoing it. It writes to out only if it can't open the terminal. It stops
// waiting when ctx ends.
func ReadSecret(ctx context.Context, in *os.File, out io.Writer, prompt string) (string, error) {
	// Like sudo and ssh, because stderr may go into a log file while the user
	// types at the terminal.
	tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err == nil {
		defer func() {
			_ = tty.Close()
		}()

		out = tty
	}

	fd := int(in.Fd()) //nolint:gosec // A file descriptor fits in an int.

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
