// Package command defines the tidegate command and its subcommands.
package command

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/tidegate-dev/tidegate/cli/internal/authority"
	"github.com/tidegate-dev/tidegate/cli/internal/output"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

// defaultServer matches the authority's default listen address.
const defaultServer = "https://127.0.0.1:8080"

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
			dummyCommand(),
		},
	}

	enableSuggestions(root)

	return root
}

func dialAuthorityUnauthenticated(cmd *cli.Command) (*authority.Clients, error) {
	return authority.Connect(cmd.String("server"), secret.New(""))
}

func dialAuthorityAuthenticated(cmd *cli.Command) (*authority.Clients, error) {
	return authority.Connect(cmd.String("server"), secret.New(os.Getenv("TIDEGATE_TOKEN")))
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
