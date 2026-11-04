package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/tidegate-dev/tidegate/authority/internal/password"
	"github.com/tidegate-dev/tidegate/internal/secret"
	"github.com/tidegate-dev/tidegate/internal/terminal"
)

var errPasswordsDiffer = errors.New("the passwords differ")

// readPassword asks for a password twice on a terminal. Otherwise it reads
// the first line from in, so a script can pipe the password in.
func readPassword(ctx context.Context, in io.Reader, out io.Writer) (secret.String, error) {
	file, ok := terminal.File(in)
	if !ok {
		line, err := readPasswordLine(ctx, in)

		return secret.New(line), err
	}

	return askPasswordTwice(func(prompt string) (string, error) {
		return terminal.ReadSecret(ctx, file, out, prompt)
	})
}

func askPasswordTwice(ask func(prompt string) (string, error)) (secret.String, error) {
	first, err := ask(fmt.Sprintf("Password, at least %d characters: ", password.MinLength))
	if err != nil {
		return secret.String{}, err
	}

	second, err := ask("Repeat the password: ")
	if err != nil {
		return secret.String{}, err
	}

	if !password.Equivalent(secret.New(first), secret.New(second)) {
		return secret.String{}, errPasswordsDiffer
	}

	return secret.New(first), nil
}

func readPasswordLine(ctx context.Context, in io.Reader) (string, error) {
	type readResult struct {
		line string
		err  error
	}

	readResultChan := make(chan readResult, 1)

	// A read from a pipe can't be canceled, so on ctx's end it's left
	// running.
	go func() {
		// Room for the longest password, a CRLF line ending and one more byte,
		// so a longer password is refused as too long instead of cut off.
		line, err := bufio.NewReader(io.LimitReader(in, password.MaxBytes+3)).ReadString('\n')
		readResultChan <- readResult{line, err}
	}()

	select {
	case result := <-readResultChan:
		if result.err != nil && !errors.Is(result.err, io.EOF) {
			return "", fmt.Errorf("reading the password from stdin: %w", result.err)
		}

		return strings.TrimSuffix(strings.TrimSuffix(result.line, "\n"), "\r"), nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
