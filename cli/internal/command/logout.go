package command

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"

	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

func logoutCommand() *cli.Command {
	return &cli.Command{
		Name:   "logout",
		Usage:  "revoke the session token in TIDEGATE_TOKEN",
		Action: logout,
	}
}

func logout(ctx context.Context, cmd *cli.Command) error {
	server := cmd.String("server")

	clients, err := dialAuthority(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	_, err = clients.Login().Logout(ctx, connect.NewRequest(&authorityv1.LogoutRequest{}))
	if err != nil {
		return fmt.Errorf("signing out at %s: %w", server, err)
	}

	return nil
}
