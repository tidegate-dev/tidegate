package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/tidegate-dev/tidegate/authority/internal/password"
	"github.com/tidegate-dev/tidegate/internal/secret"
	"github.com/tidegate-dev/tidegate/internal/terminal"
)

var errPasswordsDiffer = errors.New("the passwords differ")

// readPassword asks for a password twice on a terminal. Otherwise it reads
// the first line from in, so a script can pipe the password in.
func readPassword(ctx context.Context, in io.Reader, out io.Writer) (secret.String, error) {
	tty := terminal.New(in, out)
	if !tty.Interactive() {
		line, err := tty.ReadLine(ctx)
		if errors.Is(err, io.EOF) {
			return secret.String{}, errors.New("no password on stdin")
		}

		if err != nil {
			return secret.String{}, fmt.Errorf("reading the password from stdin: %w", err)
		}

		return secret.New(line), nil
	}

	return askPasswordTwice(func(prompt string) (string, error) {
		return tty.ReadSecret(ctx, prompt)
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
