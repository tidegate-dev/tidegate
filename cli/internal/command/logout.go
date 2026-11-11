package command

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"

	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1/authorityv1connect"
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

	err := checkServer(server)
	if err != nil {
		return err
	}

	client := authorityv1connect.NewLoginServiceClient(http.DefaultClient, server)

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	request := connect.NewRequest(&authorityv1.LogoutRequest{})
	setToken(request.Header())

	_, err = client.Logout(ctx, request)
	if err != nil {
		return fmt.Errorf("signing out at %s: %w", server, err)
	}

	return nil
}
