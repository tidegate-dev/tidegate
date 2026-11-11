// Package command defines the tidegate command and its subcommands.
package command

import (
	"context"
	"fmt"
	"os"
	"time"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"

	"github.com/tidegate-dev/tidegate/cli/internal/authority"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/internal/secret"
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
			{
				Name:   "ping",
				Usage:  "check that the authority can be reached",
				Action: ping,
			},
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

func ping(ctx context.Context, cmd *cli.Command) error {
	server := cmd.String("server")

	clients, err := authority.Connect(server, secret.New(os.Getenv("TIDEGATE_TOKEN")))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	response, err := clients.Ping().Ping(ctx, connect.NewRequest(&authorityv1.PingRequest{}))
	if connect.CodeOf(err) == connect.CodeUnavailable {
		return fmt.Errorf("can't reach the authority at %s: %w", server, err)
	}

	if err != nil {
		return fmt.Errorf("pinging the authority at %s: %w", server, err)
	}

	_, err = fmt.Fprintln(cmd.Root().Writer, response.Msg.GetMessage())

	return err
}
