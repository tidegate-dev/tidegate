package authority

import (
	"context"
	"time"

	"connectrpc.com/connect"
)

type timeoutInterceptor struct {
	limit time.Duration
}

func (i timeoutInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
		ctx, cancel := context.WithTimeout(ctx, i.limit)
		defer cancel()

		return next(ctx, request)
	}
}

func (i timeoutInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i timeoutInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}
