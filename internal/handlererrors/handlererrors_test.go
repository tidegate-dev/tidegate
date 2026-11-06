package handlererrors_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/internal/handlererrors"
	"github.com/tidegate-dev/tidegate/internal/logging"
)

func TestUnexpectedErrorReachesTheClientAsInternal(t *testing.T) {
	t.Parallel()

	// arrange
	handler := handlererrors.Interceptor().WrapUnary(failWith(errors.New("reading the text: connection refused")))

	// act
	_, err := handler(t.Context(), connect.NewRequest(&struct{}{}))

	// assert
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	assert.Equal(t, connect.CodeInternal, connectErr.Code())
	assert.Equal(t, "internal error", connectErr.Message())
}

func TestUnexpectedErrorIsLogged(t *testing.T) {
	t.Parallel()

	// arrange
	var out bytes.Buffer

	ctx := logging.NewContext(t.Context(), logging.New(&out, slog.LevelInfo, logging.FormatText))
	handler := handlererrors.Interceptor().WrapUnary(failWith(errors.New("reading the text: connection refused")))

	// act
	_, _ = handler(ctx, connect.NewRequest(&struct{}{}))

	// assert
	assert.Contains(t, out.String(), "reading the text: connection refused")
}

func TestConnectErrorPassesThrough(t *testing.T) {
	t.Parallel()

	// arrange
	notFound := connect.NewError(connect.CodeNotFound, errors.New("no text with ID 7"))
	handler := handlererrors.Interceptor().WrapUnary(failWith(notFound))

	// act
	_, err := handler(t.Context(), connect.NewRequest(&struct{}{}))

	// assert
	assert.Same(t, notFound, err)
}

func TestCanceledContextPassesThrough(t *testing.T) {
	t.Parallel()

	// arrange
	handler := handlererrors.Interceptor().WrapUnary(failWith(fmt.Errorf("storing the text: %w", context.Canceled)))

	// act
	_, err := handler(t.Context(), connect.NewRequest(&struct{}{}))

	// assert
	assert.ErrorIs(t, err, context.Canceled)
}

func TestExceededDeadlinePassesThrough(t *testing.T) {
	t.Parallel()

	// arrange
	handler := handlererrors.Interceptor().WrapUnary(failWith(fmt.Errorf("storing the text: %w", context.DeadlineExceeded)))

	// act
	_, err := handler(t.Context(), connect.NewRequest(&struct{}{}))

	// assert
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestUnexpectedStreamingErrorReachesTheClientAsInternal(t *testing.T) {
	t.Parallel()

	// arrange
	handler := handlererrors.Interceptor().WrapStreamingHandler(streamFailWith(errors.New("reading the session: connection refused")))

	// act
	err := handler(t.Context(), stream{})

	// assert
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	assert.Equal(t, connect.CodeInternal, connectErr.Code())
	assert.Equal(t, "internal error", connectErr.Message())
}

func TestUnexpectedStreamingErrorIsLogged(t *testing.T) {
	t.Parallel()

	// arrange
	var out bytes.Buffer

	ctx := logging.NewContext(t.Context(), logging.New(&out, slog.LevelInfo, logging.FormatText))
	handler := handlererrors.Interceptor().WrapStreamingHandler(streamFailWith(errors.New("reading the session: connection refused")))

	// act
	_ = handler(ctx, stream{})

	// assert
	assert.Contains(t, out.String(), "reading the session: connection refused")
}

func TestStreamingConnectErrorPassesThrough(t *testing.T) {
	t.Parallel()

	// arrange
	notFound := connect.NewError(connect.CodeNotFound, errors.New("no session with ID 7"))
	handler := handlererrors.Interceptor().WrapStreamingHandler(streamFailWith(notFound))

	// act
	err := handler(t.Context(), stream{})

	// assert
	assert.Same(t, notFound, err)
}

func failWith(err error) connect.UnaryFunc {
	return func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, err
	}
}

func streamFailWith(err error) connect.StreamingHandlerFunc {
	return func(context.Context, connect.StreamingHandlerConn) error {
		return err
	}
}

// stream sets the procedure that connect-go fills in for a real call. The
// interceptor reads nothing else from it.
type stream struct {
	connect.StreamingHandlerConn
}

func (stream) Spec() connect.Spec {
	return connect.Spec{Procedure: "/tidegate.authority.v1.SessionService/Watch"}
}
