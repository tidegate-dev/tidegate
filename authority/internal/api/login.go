package api

import (
	"context"
	"net/http"
	"slices"
	"strings"

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
	token, ok := bearerToken(req.Header())
	if !ok {
		return nil, clientError(service.ErrNotSignedIn)
	}

	// TODO(#105): take the user from the request context, where the token
	// check puts it, and drop it from what Logout returns.
	_, _, err := s.logins.Logout(ctx, token, req.Msg.GetAllSessions())
	if err != nil {
		return nil, clientError(err)
	}

	return connect.NewResponse(&authorityv1.LogoutResponse{}), nil
}

// bearerToken returns the token from an `Authorization: Bearer` header. HTTP
// compares auth schemes without regard to case.
func bearerToken(header http.Header) (secret.String, bool) {
	scheme, token, found := strings.Cut(header.Get("Authorization"), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return secret.String{}, false
	}

	return secret.New(token), true
}
