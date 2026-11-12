package command

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"

	"github.com/tidegate-dev/tidegate/cli/internal/authority"
	"github.com/tidegate-dev/tidegate/cli/internal/connections"
)

func logoutCommand() *cli.Command {
	return &cli.Command{
		Name:  "logout",
		Usage: "end the session and keep the connection",
		Description: "Ends the session at the authority and removes it from the saved connection. The\n" +
			"connection stays, so tidegate login --connection <name> signs in to it again\n" +
			"without a URL. To remove the connection as well, run tidegate connections remove.",
		Action: logout,
	}
}

func logout(ctx context.Context, cmd *cli.Command) error {
	chosen, err := pickTargetForCommand(cmd)
	if err != nil {
		return err
	}

	usesConnection := chosen.connection != ""
	if usesConnection && chosen.token.Reveal() == "" {
		_, err = fmt.Fprintf(cmd.Root().ErrWriter, "%q is already signed out\n", chosen.connection)

		return err
	}

	err = authority.SignOut(ctx, chosen.server, chosen.token)
	if usesConnection && connect.CodeOf(err) == connect.CodeUnauthenticated {
		err = nil
	}

	if err != nil && usesConnection {
		return &callError{
			action: "signing out",
			target: chosen,
			err:    err,
			hint: fmt.Sprintf(
				"The session of %q is still saved. Run `tidegate logout --connection %q` again later, or remove the connection with `tidegate connections remove %q`",
				chosen.connection, chosen.connection, chosen.connection,
			),
		}
	}

	if err != nil {
		return chosen.newCallError("signing out", err)
	}

	if !usesConnection {
		return nil
	}

	path, err := connections.DefaultPath()
	if err != nil {
		return err
	}

	return connections.SignOut(path, chosen.connection, chosen.token)
}
