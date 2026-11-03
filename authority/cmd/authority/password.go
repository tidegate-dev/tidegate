package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/tidegate-dev/tidegate/authority/internal/password"
	"github.com/tidegate-dev/tidegate/internal/terminal"
)

var errPasswordsDiffer = errors.New("the passwords differ")

// newPassword asks for a password twice on a terminal. Otherwise it reads
// the first line from in, so a script can pipe the password in.
func newPassword(ctx context.Context, in io.Reader, out io.Writer) (string, error) {
	file, ok := terminal.File(in)
	if !ok {
		return passwordLine(in)
	}

	return confirmedPassword(func(prompt string) (string, error) {
		return terminal.ReadSecret(ctx, file, out, prompt)
	})
}

func confirmedPassword(ask func(prompt string) (string, error)) (string, error) {
	first, err := ask(fmt.Sprintf("Password, at least %d characters: ", password.MinLength))
	if err != nil {
		return "", err
	}

	second, err := ask("Repeat the password: ")
	if err != nil {
		return "", err
	}

	if first != second {
		return "", errPasswordsDiffer
	}

	return first, nil
}

func passwordLine(in io.Reader) (string, error) {
	// Room for the longest password, a CRLF line ending and one more byte, so
	// a longer password is refused as too long instead of cut off.
	line, err := bufio.NewReader(io.LimitReader(in, password.MaxBytes+3)).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("reading the password from stdin: %w", err)
	}

	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}
