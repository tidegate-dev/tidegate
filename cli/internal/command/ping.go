package command

import (
	"context"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"

	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

type pingResult struct {
	Message string `json:"message" yaml:"message"`
}

func (r pingResult) String() string { return r.Message }

func pingCommand() *cli.Command {
	return &cli.Command{
		Name:   "ping",
		Usage:  "check that the authority can be reached",
		Action: ping,
	}
}

func ping(ctx context.Context, cmd *cli.Command) error {
	clients, chosen, err := connectToTarget(cmd)
	if err != nil {
		return err
	}

	response, err := clients.Ping().Ping(ctx, connect.NewRequest(&authorityv1.PingRequest{}))
	if err != nil {
		return chosen.newCallError("pinging the authority", err)
	}

	return printResult(cmd, pingResult{Message: response.Msg.GetMessage()})
}
