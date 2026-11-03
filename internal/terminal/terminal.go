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

// ReadSecret writes prompt to out and reads a line from the terminal in
// without showing what the user types. If ctx ends first, it turns the
// terminal's echo back on and returns ctx's error. The read itself can't be
// canceled and goes on in the background, so call ReadSecret only in a
// process that exits soon after a canceled read.
func ReadSecret(ctx context.Context, in *os.File, out io.Writer, prompt string) (string, error) {
	fd := int(in.Fd()) //nolint:gosec // A file descriptor fits in an int.

	state, err := term.GetState(fd)
	if err != nil {
		return "", fmt.Errorf("reading the terminal's state: %w", err)
	}

	_, err = fmt.Fprint(out, prompt)
	if err != nil {
		return "", err
	}

	type readResult struct {
		secret []byte
		err    error
	}

	readResultChan := make(chan readResult, 1)

	go func() {
		secret, err := term.ReadPassword(fd)
		readResultChan <- readResult{secret, err}
	}()

	var result readResult

	select {
	case result = <-readResultChan:
	case <-ctx.Done():
		result.err = ctx.Err()

		_ = term.Restore(fd, state)
	}

	// The user's Enter isn't shown either, so the next output would follow
	// the prompt on the same line.
	_, _ = fmt.Fprintln(out)

	if result.err != nil {
		return "", fmt.Errorf("reading from the terminal: %w", result.err)
	}

	return string(result.secret), nil
}
