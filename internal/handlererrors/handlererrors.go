// Package handlererrors keeps unexpected handler errors away from API clients.
package handlererrors

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/tidegate-dev/tidegate/internal/logging"
)

// Interceptor returns an interceptor that logs any error a unary handler
// returns without a connect code and answers the client with the code
// internal and a generic message instead. Connect errors, context.Canceled
// and context.DeadlineExceeded pass through unchanged.
func Interceptor() connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			res, err := next(ctx, req)
			if err == nil || passesThrough(err) {
				return res, err
			}

			logging.FromContext(ctx).ErrorContext(ctx, "handling a request",
				"procedure", req.Spec().Procedure,
				"error", err)

			return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
		}
	})
}

// connect itself turns context.Canceled and context.DeadlineExceeded into the
// codes canceled and deadline_exceeded.
func passesThrough(err error) bool {
	var connectErr *connect.Error

	return errors.As(err, &connectErr) ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded)
}
