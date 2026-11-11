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

func logoutCommand() *cli.Command {
	return &cli.Command{
		Name:   "logout",
		Usage:  "revoke the session token in TIDEGATE_TOKEN",
		Action: logout,
	}
}

func logout(ctx context.Context, cmd *cli.Command) error {
	server := cmd.String("server")

	clients, err := authority.Connect(server, secret.New(os.Getenv("TIDEGATE_TOKEN")))
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
