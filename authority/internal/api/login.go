package api

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/tidegate-dev/tidegate/authority/internal/password"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/internal/logging"
)

// MaxLoginRequestBytes bounds a LoginRequest. It fits a user name and a password
// of password.MaxBytes with room to spare.
const MaxLoginRequestBytes = 4096

// LoginService signs users in.
type LoginService struct {
	logins *service.Logins
}

// NewLoginService returns a LoginService that signs users in with logins.
func NewLoginService(logins *service.Logins) *LoginService {
	return &LoginService{logins: logins}
}

// Login checks the user's local password and returns a new login token.
func (s *LoginService) Login(
	ctx context.Context,
	req *connect.Request[authorityv1.LoginRequest],
) (*connect.Response[authorityv1.LoginResponse], error) {
	logger := logging.FromContext(ctx).With("user", req.Msg.GetUserName())

	token, err := s.logins.Login(ctx, req.Msg.GetUserName(), req.Msg.GetPassword())
	if err != nil {
		logger.WarnContext(ctx, "login failed", "reason", err)

		return nil, clientError(err)
	}

	logger.InfoContext(ctx, "login succeeded")

	return connect.NewResponse(&authorityv1.LoginResponse{Token: token}), nil
}

// clientError leaves out the reason of wrong credentials, so a client can't
// tell which user names exist.
func clientError(err error) error {
	switch {
	case errors.Is(err, service.ErrWrongCredentials):
		return connect.NewError(connect.CodeUnauthenticated, service.ErrWrongCredentials)
	case errors.Is(err, password.ErrTooLong):
		return connect.NewError(connect.CodeInvalidArgument, password.ErrTooLong)
	case errors.Is(err, service.ErrBusy):
		return connect.NewError(connect.CodeUnavailable, service.ErrBusy)
	default:
		return err
	}
}
