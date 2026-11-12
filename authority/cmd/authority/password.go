package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/tidegate-dev/tidegate/authority/internal/password"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

var errPasswordsDiffer = errors.New("the passwords differ")

// passwordInput is where readPassword reads from: a terminal that a person
// types at, or a pipe.
type passwordInput interface {
	Interactive() bool
	ReadLine(ctx context.Context) (string, error)
	ReadSecret(ctx context.Context, prompt string) (string, error)
}

// readPassword asks for a password twice on a terminal. Otherwise it reads
// the first line from in, so a script can pipe the password in.
func readPassword(ctx context.Context, in passwordInput) (secret.String, error) {
	if in.Interactive() {
		return readPasswordInteractive(ctx, in)
	}

	return readPasswordPiped(ctx, in)
}

func readPasswordPiped(ctx context.Context, in passwordInput) (secret.String, error) {
	line, err := in.ReadLine(ctx)
	if errors.Is(err, io.EOF) {
		return secret.String{}, errors.New("no password on stdin")
	}

	if err != nil {
		return secret.String{}, fmt.Errorf("reading the password from stdin: %w", err)
	}

	return secret.New(line), nil
}

func readPasswordInteractive(ctx context.Context, in passwordInput) (secret.String, error) {
	first, err := in.ReadSecret(ctx, fmt.Sprintf("Password, at least %d characters: ", password.MinLength))
	if err != nil {
		return secret.String{}, err
	}

	second, err := in.ReadSecret(ctx, "Repeat the password: ")
	if err != nil {
		return secret.String{}, err
	}

	if !password.Equivalent(secret.New(first), secret.New(second)) {
		return secret.String{}, errPasswordsDiffer
	}

	return secret.New(first), nil
}
