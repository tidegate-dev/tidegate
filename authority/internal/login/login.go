// Package login implements the LoginService, which signs users in to the
// authority with a local password.
package login

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/tidegate-dev/tidegate/authority/internal/logintoken"
	"github.com/tidegate-dev/tidegate/authority/internal/password"
	"github.com/tidegate-dev/tidegate/authority/internal/users"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/internal/logging"
)

// MaxRequestBytes bounds a LoginRequest. It fits a user name and a password
// of password.MaxBytes with room to spare.
const MaxRequestBytes = 4096

// Service signs users in.
type Service struct {
	authenticator *users.Authenticator
	tokens        *logintoken.Store
}

// NewService returns a Service that checks passwords with authenticator and
// hands out tokens from tokens.
func NewService(authenticator *users.Authenticator, tokens *logintoken.Store) *Service {
	return &Service{authenticator: authenticator, tokens: tokens}
}

// Login checks the user's local password and returns a new login token.
func (s *Service) Login(
	ctx context.Context,
	req *connect.Request[authorityv1.LoginRequest],
) (*connect.Response[authorityv1.LoginResponse], error) {
	logger := logging.FromContext(ctx).With("user", req.Msg.GetUserName())

	userID, err := s.authenticator.Authenticate(ctx, req.Msg.GetUserName(), req.Msg.GetPassword())
	if err != nil {
		logger.WarnContext(ctx, "login failed", "reason", err)

		return nil, clientError(err)
	}

	token, err := s.tokens.Issue(ctx, userID)
	if err != nil {
		logger.WarnContext(ctx, "login failed", "reason", err)

		return nil, err
	}

	logger.InfoContext(ctx, "login succeeded")

	return connect.NewResponse(&authorityv1.LoginResponse{Token: token}), nil
}

// clientError leaves out the reason of wrong credentials, so a client can't
// tell which user names exist.
func clientError(err error) error {
	switch {
	case errors.Is(err, users.ErrWrongCredentials):
		return connect.NewError(connect.CodeUnauthenticated, users.ErrWrongCredentials)
	case errors.Is(err, password.ErrTooLong):
		return connect.NewError(connect.CodeInvalidArgument, password.ErrTooLong)
	case errors.Is(err, users.ErrBusy):
		return connect.NewError(connect.CodeUnavailable, users.ErrBusy)
	default:
		return err
	}
}
