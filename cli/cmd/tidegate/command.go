package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"

	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1/authorityv1connect"
)

// defaultServer matches the authority's default listen address.
const defaultServer = "https://127.0.0.1:8080"

const requestTimeout = 10 * time.Second

func newCommand() *cli.Command {
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
	client := authorityv1connect.NewPingServiceClient(http.DefaultClient, server)

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	response, err := client.Ping(ctx, connect.NewRequest(&authorityv1.PingRequest{}))
	if connect.CodeOf(err) == connect.CodeUnavailable {
		return fmt.Errorf("can't reach the authority at %s: %w", server, err)
	}

	if err != nil {
		return fmt.Errorf("pinging the authority at %s: %w", server, err)
	}

	_, err = fmt.Fprintln(cmd.Root().Writer, response.Msg.GetMessage())

	return err
}
