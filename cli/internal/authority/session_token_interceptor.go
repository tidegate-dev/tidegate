package authority

import (
	"context"

	"connectrpc.com/connect"

	"github.com/tidegate-dev/tidegate/internal/secret"
)

type sessionTokenInterceptor struct {
	token secret.String
}

func (i sessionTokenInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
		request.Header().Set("Authorization", i.authorization())

		return next(ctx, request)
	}
}

func (i sessionTokenInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		conn.RequestHeader().Set("Authorization", i.authorization())

		return conn
	}
}

func (i sessionTokenInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

func (i sessionTokenInterceptor) authorization() string {
	return "Bearer " + i.token.Reveal()
}
