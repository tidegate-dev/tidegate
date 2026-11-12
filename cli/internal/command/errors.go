package command

import (
	"errors"
	"fmt"

	"connectrpc.com/connect"
)

type callError struct {
	action string
	target target
	err    error
	hint   string
}

func (e *callError) Error() string {
	return fmt.Sprintf("%s at %s: %s", e.action, quoteIfUnprintable(e.target.server), e.err)
}

func (e *callError) Unwrap() error { return e.err }

// Message returns the text that tidegate shows when a command ends with err.
func Message(err error) string {
	callErr, called := errors.AsType[*callError](err)
	if !called {
		return err.Error()
	}

	message := callErr.message()
	if callErr.hint != "" {
		message += ". " + callErr.hint
	}

	return message
}

func (e *callError) message() string {
	connectErr, connected := errors.AsType[*connect.Error](e.err)
	unauthenticated := connect.CodeOf(e.err) == connect.CodeUnauthenticated

	switch {
	case unauthenticated && e.target.fromEnvironment && e.target.token.Reveal() != "":
		return "the authority doesn't accept the session token in TIDEGATE_TOKEN. Get a new one with `tidegate login <url> --print-token`."
	case unauthenticated && e.target.fromEnvironment:
		return "not signed in. Set TIDEGATE_TOKEN to a token from `tidegate login <url> --print-token`, or unset TIDEGATE_SERVER to use a saved connection."
	case unauthenticated:
		return fmt.Sprintf("not signed in, or the session expired. Run `tidegate login --connection %q`.", e.target.connection)
	case connect.CodeOf(e.err) == connect.CodeUnavailable && connected:
		return fmt.Sprintf("can't reach the authority at %s: %s", quoteIfUnprintable(e.target.server), connectErr.Message())
	default:
		return e.Error()
	}
}
