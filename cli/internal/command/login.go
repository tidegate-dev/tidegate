package command

import (
	"context"
	"errors"
	"fmt"
	"io"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"

	"github.com/tidegate-dev/tidegate/cli/internal/authority"
	"github.com/tidegate-dev/tidegate/cli/internal/connections"
	"github.com/tidegate-dev/tidegate/internal/secret"
	"github.com/tidegate-dev/tidegate/internal/terminal"
)

type loginResult struct {
	Token string `json:"token" yaml:"token"`
}

func (result loginResult) String() string { return result.Token }

func loginCommand(store connections.Store) *cli.Command {
	return &cli.Command{
		Name:      "login",
		Usage:     "sign in to an authority and save it as a connection",
		ArgsUsage: "[<url>]",
		Description: "With a URL, signs in to the authority there and saves it as a connection. The\n" +
			"connection gets the name from --connection, or the URL if you give no name.\n" +
			"TIDEGATE_CONNECTION doesn't name a new connection. If you give no name and one\n" +
			"saved connection already has this URL, login signs in to that connection again.\n\n" +
			"Without a URL, signs in again to a saved connection: the one from --connection\n" +
			"or TIDEGATE_CONNECTION, or the only one that is saved.",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "print-token",
				Usage: "print the session token and save nothing, needs a URL and takes no --connection",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return login(ctx, cmd, store)
		},
	}
}

func login(ctx context.Context, cmd *cli.Command, store connections.Store) error {
	if cmd.Args().Len() > 1 {
		return errors.New("want at most one URL")
	}

	url := cmd.Args().First()
	connectionFlag := cmd.String("connection")

	if cmd.Bool("print-token") {
		return loginAndPrintToken(ctx, cmd, url, connectionFlag)
	}

	connection, err := store.ForLogin(url, connectionFlag)
	if err != nil {
		return err
	}

	connection.Token, err = promptAndSignIn(ctx, cmd, connection.Server)
	if err != nil {
		return err
	}

	replaced, err := store.Save(connection)
	if err != nil {
		if endSession(ctx, connection) != nil {
			return fmt.Errorf(
				"%w. The new session at %s couldn't be ended and stays valid until it expires",
				err, quoteIfUnprintable(connection.Server),
			)
		}

		return err
	}

	err = endSession(ctx, replaced)
	if err != nil {
		printWarning(cmd, "signed in, but the session that %q had before is still valid: %s", connection.Name, Message(err))
	}

	return nil
}

func loginAndPrintToken(ctx context.Context, cmd *cli.Command, url, connectionFlag string) error {
	if url == "" {
		return errors.New("--print-token needs a URL: tidegate login <url> --print-token")
	}

	if connectionFlag != "" {
		return errors.New("--print-token saves nothing, so it takes no --connection")
	}

	token, err := promptAndSignIn(ctx, cmd, url)
	if err != nil {
		return err
	}

	return printResult(cmd, loginResult{Token: token.Reveal()})
}

func promptAndSignIn(ctx context.Context, cmd *cli.Command, server string) (secret.String, error) {
	err := authority.CheckServer(server)
	if err != nil {
		return secret.String{}, err
	}

	_, err = fmt.Fprintf(cmd.Root().ErrWriter, "Signing in at %s\n", quoteIfUnprintable(server))
	if err != nil {
		return secret.String{}, err
	}

	userName, password, err := readCredentials(ctx, cmd.Root().Reader, cmd.Root().ErrWriter)
	if err != nil {
		return secret.String{}, err
	}

	token, err := authority.SignIn(ctx, server, userName, secret.New(password))
	if connect.CodeOf(err) == connect.CodeUnauthenticated {
		return secret.String{}, fmt.Errorf("signing in at %s: wrong user name or password", quoteIfUnprintable(server))
	}

	if err != nil {
		return secret.String{}, callFailed("signing in", connections.Connection{Server: server}, err)
	}

	return token, nil
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
