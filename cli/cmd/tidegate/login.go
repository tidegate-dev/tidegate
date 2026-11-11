package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"

	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1/authorityv1connect"
	"github.com/tidegate-dev/tidegate/internal/terminal"
)

func loginCommand() *cli.Command {
	return &cli.Command{
		Name:   "login",
		Usage:  "sign in and print a session token",
		Action: login,
	}
}

func login(ctx context.Context, cmd *cli.Command) error {
	userName, password, err := readCredentials(ctx, cmd.Root().Reader, cmd.Root().ErrWriter)
	if err != nil {
		return err
	}

	server := cmd.String("server")
	client := authorityv1connect.NewLoginServiceClient(http.DefaultClient, server)

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	response, err := client.Login(ctx, connect.NewRequest(&authorityv1.LoginRequest{
		UserName: userName,
		Password: password,
	}))
	if err != nil {
		return fmt.Errorf("signing in at %s: %w", server, err)
	}

	_, err = fmt.Fprintln(cmd.Root().Writer, response.Msg.GetToken())

	return err
}

// readCredentials asks for the user name and the password on a terminal.
// Otherwise it reads them from the first two lines of in, so a script can
// pipe them in.
func readCredentials(ctx context.Context, in io.Reader, prompts io.Writer) (string, string, error) {
	lines := bufio.NewReader(in)
	file, isTerminal := terminal.File(in)

	if isTerminal {
		_, err := fmt.Fprint(prompts, "User name: ")
		if err != nil {
			return "", "", err
		}
	}

	userName, err := readLine(lines)
	if err != nil {
		return "", "", fmt.Errorf("reading the user name: %w", err)
	}

	if isTerminal {
		password, err := terminal.ReadSecret(ctx, file, prompts, "Password: ")
		if err != nil {
			return "", "", fmt.Errorf("reading the password: %w", err)
		}

		return userName, password, nil
	}

	password, err := readLine(lines)
	if err != nil {
		return "", "", fmt.Errorf("reading the password: %w", err)
	}

	return userName, password, nil
}

func readLine(lines *bufio.Reader) (string, error) {
	line, err := lines.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}
