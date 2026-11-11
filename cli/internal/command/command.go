// Package command defines the tidegate command and its subcommands.
package command

import (
	"context"
	"fmt"
	"time"

	"github.com/urfave/cli/v3"
)

// defaultServer matches the authority's default listen address.
const defaultServer = "https://127.0.0.1:8080"

const requestTimeout = 10 * time.Second

// New returns the tidegate command with all its subcommands.
func New() *cli.Command {
	root := &cli.Command{
		Name:   "tidegate",
		Usage:  "talk to the tidegate authority",
		Action: unknownCommand,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "server",
				Usage: "URL of the authority",
				Value: defaultServer,
			},
		},
		Commands: []*cli.Command{
			pingCommand(),
			loginCommand(),
			logoutCommand(),
			dummyCommand(),
		},
	}

	enableSuggestions(root)

	return root
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
