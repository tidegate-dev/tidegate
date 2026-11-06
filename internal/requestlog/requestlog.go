// Package requestlog logs every call to a connect API.
package requestlog

import (
	"context"
	"errors"
	"os"
	"time"

	"connectrpc.com/connect"
	"github.com/The127/go-clock"

	"github.com/tidegate-dev/tidegate/internal/logging"
)

// Interceptor returns an interceptor that logs one line at info level for
// each unary or streaming call. The line has the procedure, the code the
// client gets, the duration and the peer address, but never a request or
// response body. List it first in connect.WithInterceptors. Otherwise it
// logs the code before other interceptors change it.
func Interceptor(clk clock.Service) connect.Interceptor {
	return interceptor{clk: clk}
}

type interceptor struct {
	clk clock.Service
}

func (i interceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		start := i.clk.Now()
		res, err := next(ctx, req)
		i.log(ctx, req.Spec(), req.Peer(), start, err)

		return res, err
	}
}

func (interceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i interceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		start := i.clk.Now()
		err := next(ctx, conn)
		i.log(ctx, conn.Spec(), conn.Peer(), start, err)

		return err
	}
}

func (i interceptor) log(ctx context.Context, spec connect.Spec, peer connect.Peer, start time.Time, err error) {
	logging.FromContext(ctx).InfoContext(ctx, "handled a call",
		"procedure", spec.Procedure,
		"code", code(err),
		"duration", i.clk.Now().Sub(start),
		"peer", peer.Addr)
}

// connect.Code has no value for success. connect-go turns a context error
// into canceled or deadline_exceeded only after the interceptors have run,
// and connect.CodeOf reports it as unknown, so code maps it the same way
// connect-go does.
func code(err error) string {
	var connectErr *connect.Error

	switch {
	case err == nil:
		return "ok"
	case errors.As(err, &connectErr):
		return connectErr.Code().String()
	case errors.Is(err, context.Canceled):
		return connect.CodeCanceled.String()
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, os.ErrDeadlineExceeded):
		return connect.CodeDeadlineExceeded.String()
	default:
		return connect.CodeUnknown.String()
	}
}
