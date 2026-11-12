package command

import (
	"errors"
	"fmt"

	"connectrpc.com/connect"

	"github.com/tidegate-dev/tidegate/cli/internal/connections"
)

// callError is a call to an authority that failed.
type callError struct {
	// action is what the command was doing, like "signing in".
	action string
	// connection is the connection that the call used.
	connection connections.Connection
	err        error
	// hint is an extra sentence that tells the user what to do next.
	hint string
}

func callFailed(action string, connection connections.Connection, err error) error {
	return &callError{action: action, connection: connection, err: err}
}

func (callErr *callError) Error() string {
	return fmt.Sprintf("%s at %s: %s", callErr.action, quoteIfUnprintable(callErr.connection.Server), callErr.err)
}

func (callErr *callError) Unwrap() error { return callErr.err }

// Message returns the text that tidegate shows when a command ends with err.
func Message(err error) string {
	callErr, isCallError := errors.AsType[*callError](err)
	if !isCallError {
		return err.Error()
	}

	message := callErr.message()
	if callErr.hint != "" {
		message += ". " + callErr.hint
	}

	return message
}

func (callErr *callError) message() string {
	connectErr, isConnectError := errors.AsType[*connect.Error](callErr.err)
	notSignedIn := connect.CodeOf(callErr.err) == connect.CodeUnauthenticated
	authorityUnreachable := connect.CodeOf(callErr.err) == connect.CodeUnavailable

	// A connection from TIDEGATE_SERVER and TIDEGATE_TOKEN has no name.
	fromEnvironment := callErr.connection.Name == ""
	hasToken := callErr.connection.Token.Reveal() != ""

	switch {
	case notSignedIn && fromEnvironment && hasToken:
		return "the authority doesn't accept the session token in TIDEGATE_TOKEN. Get a new one with `tidegate login <url> --print-token`."
	case notSignedIn && fromEnvironment:
		return "not signed in. Set TIDEGATE_TOKEN to a token from `tidegate login <url> --print-token`, or unset TIDEGATE_SERVER to use a saved connection."
	case notSignedIn:
		return fmt.Sprintf("not signed in, or the session expired. Run `tidegate login --connection %q`.", callErr.connection.Name)
	case authorityUnreachable && isConnectError:
		return fmt.Sprintf("can't reach the authority at %s: %s", quoteIfUnprintable(callErr.connection.Server), connectErr.Message())
	default:
		return callErr.Error()
	}
}
