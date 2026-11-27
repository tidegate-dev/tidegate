// Package command defines the phemera command and its subcommands.
package command

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/phemera-dev/phemera/cli/internal/authority"
	"github.com/phemera-dev/phemera/cli/internal/connections"
	"github.com/phemera-dev/phemera/cli/internal/output"
)

// New returns the phemera command with all its subcommands. The subcommands
// read and change the saved connections through store.
func New(store connections.Store) *cli.Command {
	root := &cli.Command{
		Name:  "phemera",
		Usage: "command line interface for a phemera installation",
		Description: "phemera login <url> signs in and saves the authority as a connection. With one\n" +
			"saved connection, commands use it. With several, choose one with --connection.\n\n" +
			"A script that brings its own session token sets PHEMERA_SERVER and\n" +
			"PHEMERA_TOKEN. Commands then use these and no saved connection.",
		Action: unknownCommand,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "connection",
				Aliases: []string{"c"},
				Usage:   "name of the saved connection to use, by default the one in PHEMERA_CONNECTION",
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
			loginCommand(store),
			logoutCommand(store),
			connectionsCommand(store),
			projectCommand(store),
			roleCommand(store),
			assetCommand(store),
			bindingCommand(store),
			connectCommand(store),
		},
	}

	enableSuggestions(root)
	refuseUnknownSubcommands(root)

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
	if name == "" && cmd == cmd.Root() {
		return cli.ShowRootCommandHelp(cmd)
	}

	if name == "" {
		return cli.ShowSubcommandHelp(cmd)
	}

	return fmt.Errorf("unknown command %q. Did you mean %q?", name, cli.SuggestCommand(cmd.Commands, name))
}

// urfave/cli answers an unknown subcommand of a command without an action with status 3 and no "Error:".
func refuseUnknownSubcommands(cmd *cli.Command) {
	for _, subcommand := range cmd.Commands {
		if len(subcommand.Commands) > 0 && subcommand.Action == nil {
			subcommand.Action = unknownCommand
		}

		refuseUnknownSubcommands(subcommand)
	}
}

func enableSuggestions(cmd *cli.Command) {
	cmd.Suggest = true

	// Subcommands don't inherit Suggest from their parent.
	for _, subcommand := range cmd.Commands {
		enableSuggestions(subcommand)
	}
}
