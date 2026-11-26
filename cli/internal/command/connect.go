package command

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/The127/go-random"
	"github.com/google/uuid"
	"github.com/urfave/cli/v3"

	"github.com/phemera-dev/phemera/cli/internal/connections"
	"github.com/phemera-dev/phemera/cli/internal/sshsession"
	authorityv1 "github.com/phemera-dev/phemera/gen/phemera/authority/v1"
)

// just dev and just kind serve the ssh-proxy prototype on this address.
const sshProxyAddress = "127.0.0.1:2022"

func connectCommand(store connections.Store) *cli.Command {
	return &cli.Command{
		Name:      "connect",
		Usage:     "open a shell on an SSH asset",
		ArgsUsage: "<asset-id>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return openSession(ctx, cmd, store)
		},
	}
}

func openSession(ctx context.Context, cmd *cli.Command, store connections.Store) error {
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

	_, err = clients.Ticket().CreateTicket(ctx, connect.NewRequest(&authorityv1.CreateTicketRequest{
		AssetId: id.String(),
	}))
	if err != nil {
		return callFailed("creating the ticket", connection, err)
	}

	return sshsession.Run(ctx, random.NewCryptoRandom(), sshProxyAddress, sshsession.Streams{
		In:  cmd.Root().Reader,
		Out: cmd.Root().Writer,
		Err: cmd.Root().ErrWriter,
	})
}
