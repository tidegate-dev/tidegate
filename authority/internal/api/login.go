package api

import (
	"context"
	"net/http"
	"slices"

	"connectrpc.com/connect"

	"github.com/tidegate-dev/tidegate/authority/internal/service"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1/authorityv1connect"
	"github.com/tidegate-dev/tidegate/internal/logging"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

// maxLoginRequestBytes bounds a LoginRequest.
const maxLoginRequestBytes = 4096

// LoginService signs users in.
type LoginService struct {
	logins *service.Logins
}

// NewLoginService returns a LoginService.
func NewLoginService(logins *service.Logins) *LoginService {
	return &LoginService{logins: logins}
}

// NewLoginServiceHandler returns the path and the HTTP handler of a
// LoginService that refuses requests over maxLoginRequestBytes.
func NewLoginServiceHandler(logins *service.Logins, opts ...connect.HandlerOption) (string, http.Handler) {
	opts = slices.Concat(opts, []connect.HandlerOption{connect.WithReadMaxBytes(maxLoginRequestBytes)})

	return authorityv1connect.NewLoginServiceHandler(NewLoginService(logins), opts...)
}

// Login checks the user's local password and returns a new login token.
func (s *LoginService) Login(
	ctx context.Context,
	req *connect.Request[authorityv1.LoginRequest],
) (*connect.Response[authorityv1.LoginResponse], error) {
	logger := logging.FromContext(ctx).With("user", req.Msg.GetUserName())

	token, err := s.logins.Login(ctx, req.Msg.GetUserName(), secret.New(req.Msg.GetPassword()))
	if err != nil {
		logger.WarnContext(ctx, "login failed", "reason", err)

		return nil, clientError(err)
	}

	return connect.NewResponse(&authorityv1.LoginResponse{Token: token.Reveal()}), nil
}

// Logout revokes the caller's login token, or every login token of the
// caller's user.
func (s *LoginService) Logout(
	ctx context.Context,
	req *connect.Request[authorityv1.LogoutRequest],
) (*connect.Response[authorityv1.LogoutResponse], error) {
	session, err := sessionFromContext(ctx)
	if err != nil {
		return nil, err
	}

	err = s.logins.Logout(ctx, session, req.Msg.GetAllSessions())
	if err != nil {
		return nil, clientError(err)
	}

	return connect.NewResponse(&authorityv1.LogoutResponse{}), nil
}
