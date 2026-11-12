package command

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"

	"github.com/tidegate-dev/tidegate/cli/internal/authority"
	"github.com/tidegate-dev/tidegate/cli/internal/connections"
)

func logoutCommand(store connections.Store) *cli.Command {
	return &cli.Command{
		Name:  "logout",
		Usage: "end the session and keep the connection",
		Description: "Ends the session at the authority and removes it from the saved connection. The\n" +
			"connection stays, so tidegate login --connection <name> signs in to it again\n" +
			"without a URL. To remove the connection as well, run tidegate connections remove.",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return logout(ctx, cmd, store)
		},
	}
}

func logout(ctx context.Context, cmd *cli.Command, store connections.Store) error {
	connection, err := store.Current(cmd.String("connection"))
	if err != nil {
		return err
	}

	// A connection from TIDEGATE_SERVER and TIDEGATE_TOKEN has no name and isn't saved.
	isSaved := connection.Name != ""

	if isSaved && connection.Token.Reveal() == "" {
		_, err = fmt.Fprintf(cmd.Root().ErrWriter, "%q is already signed out\n", connection.Name)

		return err
	}

	err = authority.SignOut(ctx, connection.Server, connection.Token)

	authorityForgotSession := connect.CodeOf(err) == connect.CodeUnauthenticated
	if isSaved && authorityForgotSession {
		err = nil
	}

	if err != nil && isSaved {
		return &callError{
			action:     "signing out",
			connection: connection,
			err:        err,
			hint: fmt.Sprintf(
				"The session of %q is still saved. Run `tidegate logout --connection %q` again later, or remove the connection with `tidegate connections remove %q`",
				connection.Name, connection.Name, connection.Name,
			),
		}
	}

	if err != nil {
		return callFailed("signing out", connection, err)
	}

	if !isSaved {
		return nil
	}

	return store.SignOut(connection)
}
