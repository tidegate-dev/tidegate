package command

import (
	"context"

	"connectrpc.com/connect"

	"github.com/tidegate-dev/tidegate/cli/internal/authority"
	"github.com/tidegate-dev/tidegate/cli/internal/connections"
)

// endSession ends the session of connection at its authority, if it has one.
func endSession(ctx context.Context, connection connections.Connection) error {
	if connection.Token.Reveal() == "" {
		return nil
	}

	err := authority.SignOut(ctx, connection.Server, connection.Token)

	authorityForgotSession := connect.CodeOf(err) == connect.CodeUnauthenticated
	if err != nil && !authorityForgotSession {
		return callFailed("signing out", connection, err)
	}

	return nil
}
