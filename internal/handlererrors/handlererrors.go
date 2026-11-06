// Package handlererrors keeps unexpected handler errors away from API clients.
package handlererrors

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/tidegate-dev/tidegate/internal/logging"
)

// Interceptor returns an interceptor for unary and streaming calls that
// catches errors without a connect code. It logs such an error and sends the
// client the code internal with a generic message instead. Connect errors,
// context.Canceled and context.DeadlineExceeded pass through unchanged.
func Interceptor() connect.Interceptor {
	return interceptor{}
}

type interceptor struct{}

func (interceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		res, err := next(ctx, req)
		if err == nil || passesThrough(err) {
			return res, err
		}

		return nil, hide(ctx, req.Spec(), err)
	}
}

func (interceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (interceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		err := next(ctx, conn)
		if err == nil || passesThrough(err) {
			return err
		}

		return hide(ctx, conn.Spec(), err)
	}
}

func hide(ctx context.Context, spec connect.Spec, err error) error {
	logging.FromContext(ctx).ErrorContext(ctx, "handling a request",
		"procedure", spec.Procedure,
		"error", err)

	return connect.NewError(connect.CodeInternal, errors.New("internal error"))
}

// connect itself turns context.Canceled and context.DeadlineExceeded into the
// codes canceled and deadline_exceeded.
func passesThrough(err error) bool {
	var connectErr *connect.Error

	return errors.As(err, &connectErr) ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded)
}
