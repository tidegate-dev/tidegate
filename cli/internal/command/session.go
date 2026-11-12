package command

import (
	"context"

	"connectrpc.com/connect"

	"github.com/tidegate-dev/tidegate/cli/internal/authority"
	"github.com/tidegate-dev/tidegate/cli/internal/connections"
)

func endSession(ctx context.Context, connection connections.Connection) error {
	if connection.Token.Reveal() == "" {
		return nil
	}

	err := authority.SignOut(ctx, connection.Server, connection.Token)
	if err != nil && connect.CodeOf(err) != connect.CodeUnauthenticated {
		return target{server: connection.Server}.newCallError("signing out", err)
	}

	return nil
}
