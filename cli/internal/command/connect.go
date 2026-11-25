package command

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/urfave/cli/v3"

	"github.com/tidegate-dev/tidegate/cli/internal/connections"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

type ticketResult struct {
	ID      string `json:"id" yaml:"id"`
	AssetID string `json:"assetId" yaml:"assetId"`
	UserID  string `json:"userId" yaml:"userId"`
}

func (t ticketResult) String() string {
	return quoteIfUnprintable(t.ID) + "\t" + quoteIfUnprintable(t.AssetID) + "\t" + quoteIfUnprintable(t.UserID)
}

func connectCommand(store connections.Store) *cli.Command {
	return &cli.Command{
		Name:      "connect",
		Usage:     "request an access ticket for an asset",
		ArgsUsage: "<asset-id>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return requestTicket(ctx, cmd, store)
		},
	}
}

func requestTicket(ctx context.Context, cmd *cli.Command, store connections.Store) error {
	if cmd.Args().Len() != 1 {
		return errors.New("want exactly one asset ID")
	}

	id, err := uuid.Parse(cmd.Args().First())
	if err != nil {
		return errors.New("want an asset ID in UUID format")
	}

	clients, connection, err := connectToCurrent(cmd, store)
	if err != nil {
		return err
	}

	response, err := clients.Ticket().CreateTicket(ctx, connect.NewRequest(&authorityv1.CreateTicketRequest{
		AssetId: id.String(),
	}))
	if err != nil {
		return callFailed("creating the ticket", connection, err)
	}

	return printResult(cmd, ticketResult{
		ID:      response.Msg.GetId(),
		AssetID: response.Msg.GetAssetId(),
		UserID:  response.Msg.GetUserId(),
	})
}
