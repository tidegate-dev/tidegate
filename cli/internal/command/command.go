// Package command defines the tidegate command and its subcommands.
package command

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/tidegate-dev/tidegate/cli/internal/authority"
	"github.com/tidegate-dev/tidegate/cli/internal/connections"
	"github.com/tidegate-dev/tidegate/cli/internal/output"
)

// New returns the tidegate command with all its subcommands. The subcommands
// read and change the saved connections through store.
func New(store connections.Store) *cli.Command {
	root := &cli.Command{
		Name:  "tidegate",
		Usage: "command line interface for a tidegate installation",
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
			pingCommand(store),
			loginCommand(store),
			logoutCommand(store),
			connectionsCommand(store),
			dummyCommand(store),
		},
	}

	enableSuggestions(root)

	return root
}

// connectToCurrent returns clients for the authority of the connection that
// the command uses, and that connection.
func connectToCurrent(cmd *cli.Command, store connections.Store) (*authority.Clients, connections.Connection, error) {
	connection, err := store.Current(cmd.String("connection"))
	if err != nil {
		return nil, connections.Connection{}, err
	}

	clients, err := authority.Connect(connection.Server, connection.Token)
	if err != nil {
		return nil, connections.Connection{}, err
	}

	return clients, connection, nil
}

func printResult(cmd *cli.Command, result output.Result) error {
	return output.Print(cmd.Root().Writer, output.Format(cmd.String("output")), result)
}

func printWarning(cmd *cli.Command, format string, arguments ...any) {
	_, _ = fmt.Fprintf(cmd.Root().ErrWriter, "Warning: "+format+"\n", arguments...)
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
	for _, subcommand := range cmd.Commands {
		enableSuggestions(subcommand)
	}
}
