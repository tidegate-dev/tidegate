// Package handlererrors hides the text of handler errors from API clients.
package handlererrors

import (
	"context"
	"errors"
	"os"

	"connectrpc.com/connect"

	"github.com/tidegate-dev/tidegate/internal/logging"
)

// Interceptor returns an interceptor that logs any handler error except a
// connect error, and sends the client a generic one with the matching code.
func Interceptor() connect.Interceptor {
	return interceptor{}
}

type interceptor struct{}

func (interceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		res, err := next(ctx, req)
		if err == nil {
			return res, nil
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
		if err == nil {
			return nil
		}

		return hide(ctx, conn.Spec(), err)
	}
}

// connect-go sends the text of any error but a connect error to the client,
// even a context error.
func hide(ctx context.Context, spec connect.Spec, err error) error {
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		return err
	}

	logger := logging.FromContext(ctx).With("procedure", spec.Procedure, "error", err)

	switch {
	case errors.Is(err, context.Canceled):
		logger.DebugContext(ctx, "the call was cancelled")

		return connect.NewError(connect.CodeCanceled, context.Canceled)
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, os.ErrDeadlineExceeded):
		if ctx.Err() == nil {
			logger.WarnContext(ctx, "a timeout inside the server ended the call")
		} else {
			logger.DebugContext(ctx, "the call timed out")
		}

		return connect.NewError(connect.CodeDeadlineExceeded, context.DeadlineExceeded)
	default:
		logger.ErrorContext(ctx, "handling a request")

		return connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
}
