package api

import (
	"errors"

	"connectrpc.com/connect"

	"github.com/tidegate-dev/tidegate/authority/internal/password"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
)

// clientError gives a service error the code and the fixed message that a
// client sees. The reason stays out, so clients can't tell which user names
// exist or why a token was refused.
func clientError(err error) error {
	switch {
	case errors.Is(err, service.ErrWrongCredentials):
		return connect.NewError(connect.CodeUnauthenticated, service.ErrWrongCredentials)
	case errors.Is(err, service.ErrNotSignedIn):
		return connect.NewError(connect.CodeUnauthenticated, service.ErrNotSignedIn)
	case errors.Is(err, password.ErrTooLong):
		return connect.NewError(connect.CodeInvalidArgument, password.ErrTooLong)
	case errors.Is(err, service.ErrBusy):
		return connect.NewError(connect.CodeUnavailable, service.ErrBusy)
	default:
		return err
	}
}
