package command

import (
	"context"
	"errors"
	"fmt"
	"io"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"

	"github.com/tidegate-dev/tidegate/cli/internal/authority"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/internal/secret"
	"github.com/tidegate-dev/tidegate/internal/terminal"
)

type loginResult struct {
	Token string `json:"token" yaml:"token"`
}

func (r loginResult) String() string { return r.Token }

func loginCommand() *cli.Command {
	return &cli.Command{
		Name:   "login",
		Usage:  "sign in and print a session token",
		Action: login,
	}
}

func login(ctx context.Context, cmd *cli.Command) error {
	server := cmd.String("server")

	clients, err := authority.Connect(server, secret.String{})
	if err != nil {
		return err
	}

	userName, password, err := readCredentials(ctx, cmd.Root().Reader, cmd.Root().ErrWriter)
	if err != nil {
		return err
	}

	response, err := clients.Login().Login(ctx, connect.NewRequest(&authorityv1.LoginRequest{
		UserName: userName,
		Password: password,
	}))
	if connect.CodeOf(err) == connect.CodeUnauthenticated {
		return fmt.Errorf("signing in at %s: wrong user name or password", server)
	}

	if err != nil {
		return fmt.Errorf("signing in at %s: %w", server, err)
	}

	return printResult(cmd, loginResult{Token: response.Msg.GetToken()})
}

// readCredentials asks for the user name and the password on a terminal.
// Otherwise it reads them from the first two lines of in, so a script can
// pipe them in.
func readCredentials(ctx context.Context, in io.Reader, prompts io.Writer) (string, string, error) {
	tty := terminal.New(in, prompts)

	err := tty.Prompt("User name: ")
	if err != nil {
		return "", "", err
	}

	userName, err := tty.ReadLine(ctx)
	if errors.Is(err, io.EOF) {
		return "", "", errors.New("no user name on stdin")
	}

	if err != nil {
		return "", "", fmt.Errorf("reading the user name: %w", err)
	}

	password, err := tty.ReadSecret(ctx, "Password: ")
	if errors.Is(err, io.EOF) {
		return "", "", errors.New("no password on stdin")
	}

	if err != nil {
		return "", "", fmt.Errorf("reading the password: %w", err)
	}

	return userName, password, nil
}
