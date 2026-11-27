package command

import (
	"errors"
	"fmt"

	"connectrpc.com/connect"

	"github.com/phemera-dev/phemera/cli/internal/connections"
	"github.com/phemera-dev/phemera/cli/internal/sshsession"
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
	return fmt.Sprintf("%s at %s: %s", callErr.action, quoteIfUnprintable(callErr.connection.Server), quoteIfUnprintable(fmt.Sprint(callErr.err)))
}

func (callErr *callError) Unwrap() error { return callErr.err }

// ssh exits with this status for its own errors.
const connectFailedStatus = 255

// connectError is an error that phemera connect ended with.
type connectError struct {
	err error
}

func connectFailed(err error) error {
	return &connectError{err: err}
}

func (connectErr *connectError) Error() string { return connectErr.err.Error() }

func (connectErr *connectError) Unwrap() error { return connectErr.err }

// ExitStatus returns the status that phemera exits with when a command ends
// with err. For phemera connect, that is the status of the shell if the shell
// ended with a failure, and 255 for every other error.
func ExitStatus(err error) int {
	shellExit, shellExited := errors.AsType[*sshsession.ExitError](err)
	if shellExited {
		return shellExit.Status
	}

	_, isConnectError := errors.AsType[*connectError](err)
	if isConnectError {
		return connectFailedStatus
	}

	return 1
}

// Message returns the text that phemera shows when a command ends with err.
// It is empty when the shell of phemera connect ended with a failure.
func Message(err error) string {
	_, shellExited := errors.AsType[*sshsession.ExitError](err)
	if shellExited {
		return ""
	}

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

	// A connection from PHEMERA_SERVER and PHEMERA_TOKEN has no name.
	fromEnvironment := callErr.connection.Name == ""
	hasToken := callErr.connection.Token.Reveal() != ""

	switch {
	case notSignedIn && fromEnvironment && hasToken:
		return "the authority doesn't accept the session token in PHEMERA_TOKEN. Get a new one with `phemera login <url> --print-token`."
	case notSignedIn && fromEnvironment:
		return "not signed in. Set PHEMERA_TOKEN to a token from `phemera login <url> --print-token`, or unset PHEMERA_SERVER to use a saved connection."
	case notSignedIn:
		return fmt.Sprintf("not signed in, or the session expired. Run `phemera login --connection %q`.", callErr.connection.Name)
	case authorityUnreachable && isConnectError:
		return fmt.Sprintf("can't reach the authority at %s: %s", quoteIfUnprintable(callErr.connection.Server), quoteIfUnprintable(connectErr.Message()))
	default:
		return callErr.Error()
	}
}
