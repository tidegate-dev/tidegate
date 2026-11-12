// Package terminal reads secrets from a terminal without showing them.
package terminal

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// Terminal reads what a person types at a terminal or what a script pipes
// in.
type Terminal struct {
	in       *os.File
	lines    *bufio.Reader
	fallback io.Writer
}

// New returns a Terminal that reads from in. The Terminal writes to fallback
// only if it can't open the terminal for writing.
func New(in io.Reader, fallback io.Writer) *Terminal {
	lines := bufio.NewReader(in)

	file, ok := in.(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) { //nolint:gosec // A file descriptor fits in an int.
		return &Terminal{lines: lines, fallback: fallback}
	}

	return &Terminal{in: file, lines: lines, fallback: fallback}
}

// Interactive reports whether the input is a terminal.
func (t *Terminal) Interactive() bool {
	return t.in != nil
}

// ReadLine reads one line without its line ending. It returns io.EOF if
// the input has ended.
func (t *Terminal) ReadLine() (string, error) {
	line, err := t.lines.ReadString('\n')
	if errors.Is(err, io.EOF) && line == "" {
		return "", io.EOF
	}

	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}

// Prompt writes text to the terminal. It writes nothing if the input isn't a
// terminal.
func (t *Terminal) Prompt(text string) error {
	if !t.Interactive() {
		return nil
	}

	out, closeOutput := t.output()
	defer closeOutput()

	_, err := fmt.Fprint(out, text)

	return err
}

// ReadSecret writes prompt to the terminal and reads a line without echoing
// it. It stops waiting when ctx ends. If the input isn't a terminal, it
// reads a line like ReadLine.
func (t *Terminal) ReadSecret(ctx context.Context, prompt string) (string, error) {
	if !t.Interactive() {
		return t.ReadLine()
	}

	fd := int(t.in.Fd()) //nolint:gosec // A file descriptor fits in an int.

	state, err := term.GetState(fd)
	if err != nil {
		return "", fmt.Errorf("reading the terminal's state: %w", err)
	}

	defer func() {
		_ = term.Restore(fd, state)
	}()

	err = t.Prompt(prompt)
	if err != nil {
		return "", err
	}

	// The Enter isn't echoed either.
	defer func() {
		_ = t.Prompt("\n")
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
