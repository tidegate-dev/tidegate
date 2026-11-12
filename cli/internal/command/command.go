// Package command defines the tidegate command and its subcommands.
package command

import (
	"context"
	"errors"
	"fmt"
	"os"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"

	"github.com/tidegate-dev/tidegate/cli/internal/authority"
	"github.com/tidegate-dev/tidegate/cli/internal/connections"
	"github.com/tidegate-dev/tidegate/cli/internal/output"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

// New returns the tidegate command with all its subcommands.
func New() *cli.Command {
	root := &cli.Command{
		Name:  "tidegate",
		Usage: "talk to a tidegate authority",
		Description: "tidegate login <url> signs in and saves the authority as a connection. With one\n" +
			"saved connection, commands use it. With several, choose one with --connection.\n\n" +
			"A script that brings its own session token sets TIDEGATE_SERVER and\n" +
			"TIDEGATE_TOKEN. Commands then use these and no saved connection.",
		Action: unknownCommand,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "connection",
				Usage: "name of the saved connection to use, by default the one in TIDEGATE_CONNECTION",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "format of the result: text, json or yaml",
				Value:   string(output.Text),
				Validator: func(value string) error {
					_, err := output.ParseFormat(value)

					return err
				},
			},
		},
		Commands: []*cli.Command{
			pingCommand(),
			loginCommand(),
			logoutCommand(),
			connectionsCommand(),
			dummyCommand(),
		},
	}

	enableSuggestions(root)

	return root
}

func connectToTarget(cmd *cli.Command) (*authority.Clients, target, error) {
	chosen, err := pickTargetForCommand(cmd)
	if err != nil {
		return nil, target{}, err
	}

	clients, err := authority.Connect(chosen.server, chosen.token)
	if err != nil {
		return nil, target{}, err
	}

	return clients, chosen, nil
}

func pickTargetForCommand(cmd *cli.Command) (target, error) {
	options := readTargetOptions(cmd)

	var byName map[string]connections.Connection

	if options.usesConnectionsFile() {
		var err error

		byName, err = loadConnections()
		if err != nil {
			return target{}, err
		}
	}

	return pickTarget(options, byName)
}

func endSession(ctx context.Context, connection connections.Connection) error {
	if connection.Token.Reveal() == "" {
		return nil
	}

	err := authority.SignOut(ctx, connection.Server, connection.Token)
	if err != nil && connect.CodeOf(err) != connect.CodeUnauthenticated {
		return target{server: connection.Server}.newCallError("signing out", err)
	}

	return nil
}

func printWarning(cmd *cli.Command, format string, arguments ...any) {
	_, _ = fmt.Fprintf(cmd.Root().ErrWriter, "Warning: "+format+"\n", arguments...)
}

func readTargetOptions(cmd *cli.Command) targetOptions {
	return targetOptions{
		connection:            cmd.String("connection"),
		environmentConnection: os.Getenv("TIDEGATE_CONNECTION"),
		server:                os.Getenv("TIDEGATE_SERVER"),
		token:                 secret.New(os.Getenv("TIDEGATE_TOKEN")),
	}
}

func loadConnections() (map[string]connections.Connection, error) {
	path, err := connections.DefaultPath()
	// A container may have no configuration directory.
	if errors.Is(err, connections.ErrNoConfigDir) {
		return map[string]connections.Connection{}, nil
	}

	if err != nil {
		return nil, err
	}

	return connections.Load(path)
}

func unknownCommand(_ context.Context, cmd *cli.Command) error {
	name := cmd.Args().First()
	if name == "" {
		return cli.ShowRootCommandHelp(cmd)
	}

	return fmt.Errorf("unknown command %q. Did you mean %q?", name, cli.SuggestCommand(cmd.Commands, name))
}

func enableSuggestions(cmd *cli.Command) {
	cmd.Suggest = true

	// Subcommands don't inherit Suggest from their parent.
	for _, sub := range cmd.Commands {
		enableSuggestions(sub)
	}
}

func printResult(cmd *cli.Command, result output.Result) error {
	return output.Print(cmd.Root().Writer, output.Format(cmd.String("output")), result)
}
