package api

import (
	"errors"

	"connectrpc.com/connect"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	"github.com/tidegate-dev/tidegate/authority/internal/password"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
)

// clientError gives a service error the code and the fixed message that a
// client sees. The reason stays out, so a failed login doesn't reveal which
// user names exist or why a token was refused.
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
	case errors.Is(err, model.ErrProjectNameTaken):
		return connect.NewError(connect.CodeAlreadyExists, model.ErrProjectNameTaken)
	case errors.Is(err, model.ErrProjectNotFound):
		return connect.NewError(connect.CodeNotFound, model.ErrProjectNotFound)
	case errors.Is(err, model.ErrRoleNameTaken):
		return connect.NewError(connect.CodeAlreadyExists, model.ErrRoleNameTaken)
	case errors.Is(err, model.ErrRoleNotFound):
		return connect.NewError(connect.CodeNotFound, model.ErrRoleNotFound)
	case errors.Is(err, model.ErrUserNotFound):
		return connect.NewError(connect.CodeNotFound, model.ErrUserNotFound)
	case errors.Is(err, model.ErrAssetNameTaken):
		return connect.NewError(connect.CodeAlreadyExists, model.ErrAssetNameTaken)
	case errors.Is(err, model.ErrBindingExists):
		return connect.NewError(connect.CodeAlreadyExists, model.ErrBindingExists)
	default:
		return err
	}
}
