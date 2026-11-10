package api

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"

	"connectrpc.com/connect"

	"github.com/tidegate-dev/tidegate/authority/internal/service"
	"github.com/tidegate-dev/tidegate/internal/logging"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

// SessionLookup finds the session that a login token belongs to.
type SessionLookup interface {
	Session(ctx context.Context, token secret.String) (service.Session, error)
}

// NewSessionInterceptor returns an interceptor that refuses a call without a
// valid login token and puts the caller's session into the context of every
// other call. The procedures in public need no login token and get no
// session.
func NewSessionInterceptor(sessions SessionLookup, public ...string) connect.Interceptor {
	return sessionInterceptor{sessions: sessions, public: public}
}

type sessionInterceptor struct {
	sessions SessionLookup
	public   []string
}

func (i sessionInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		ctx, err := i.withSession(ctx, req.Spec(), req.Header())
		if err != nil {
			return nil, err
		}

		return next(ctx, req)
	}
}

func (sessionInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i sessionInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		ctx, err := i.withSession(ctx, conn.Spec(), conn.RequestHeader())
		if err != nil {
			return err
		}

		return next(ctx, conn)
	}
}

func (i sessionInterceptor) withSession(ctx context.Context, spec connect.Spec, header http.Header) (context.Context, error) {
	if slices.Contains(i.public, spec.Procedure) {
		return ctx, nil
	}

	token, ok := bearerToken(header)
	if !ok {
		return ctx, clientError(service.ErrNotSignedIn)
	}

	session, err := i.sessions.Session(ctx, token)
	if err != nil {
		return ctx, clientError(err)
	}

	logger := logging.FromContext(ctx).With("user_id", session.UserID.String(), "session_id", session.ID.String())

	return context.WithValue(logging.NewContext(ctx, logger), sessionKey{}, session), nil
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

type sessionKey struct{}

var errNoSession = errors.New("the call has no session")

// sessionFromContext returns the session that the session interceptor found
// for the call. It returns errNoSession for a public procedure.
func sessionFromContext(ctx context.Context) (service.Session, error) {
	session, ok := ctx.Value(sessionKey{}).(service.Session)
	if !ok {
		return service.Session{}, errNoSession
	}

	return session, nil
}
