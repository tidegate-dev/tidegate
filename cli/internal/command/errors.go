package command

import (
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v3"
)

// Message returns the text that tidegate shows when root ends with err.
func Message(root *cli.Command, err error) string {
	switch connect.CodeOf(err) {
	case connect.CodeUnauthenticated:
		return "not signed in, or the session token expired. Run `tidegate login` and set TIDEGATE_TOKEN."
	case connect.CodeUnavailable:
		if connectErr, ok := errors.AsType[*connect.Error](err); ok {
			return fmt.Sprintf("can't reach the authority at %s: %s", root.String("server"), connectErr.Message())
		}

		return err.Error()
	default:
		return err.Error()
	}
}
