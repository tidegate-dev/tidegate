package api

import (
	"context"
	"net/http"
	"slices"

	"connectrpc.com/connect"

	"github.com/tidegate-dev/tidegate/authority/internal/service"
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

	return context.WithValue(ctx, sessionKey{}, session), nil
}

type sessionKey struct{}

// sessionFromContext returns the session that the session interceptor found
// for the call. A public procedure has none.
func sessionFromContext(ctx context.Context) (service.Session, bool) {
	session, ok := ctx.Value(sessionKey{}).(service.Session)

	return session, ok
}
