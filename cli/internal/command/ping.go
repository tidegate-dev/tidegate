package command

import (
	"context"
	"fmt"
	"os"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"

	"github.com/tidegate-dev/tidegate/cli/internal/authority"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

func pingCommand() *cli.Command {
	return &cli.Command{
		Name:   "ping",
		Usage:  "check that the authority can be reached",
		Action: ping,
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
