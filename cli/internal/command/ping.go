package command

import (
	"context"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"

	"github.com/tidegate-dev/tidegate/cli/internal/connections"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

type pingResult struct {
	Message string `json:"message" yaml:"message"`
}

func (result pingResult) String() string { return result.Message }

func pingCommand(store connections.Store) *cli.Command {
	return &cli.Command{
		Name:  "ping",
		Usage: "check that the authority can be reached",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return ping(ctx, cmd, store)
		},
	}
}

func ping(ctx context.Context, cmd *cli.Command, store connections.Store) error {
	clients, connection, err := connectToCurrent(cmd, store)
	if err != nil {
		return err
	}

	response, err := clients.Ping().Ping(ctx, connect.NewRequest(&authorityv1.PingRequest{}))
	if err != nil {
		return callFailed("pinging the authority", connection, err)
	}

	return printResult(cmd, pingResult{Message: response.Msg.GetMessage()})
}
