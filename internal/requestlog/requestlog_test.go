package requestlog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/The127/go-clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/internal/logging"
	"github.com/tidegate-dev/tidegate/internal/requestlog"
)

var start = time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)

func TestSuccessfulCallIsLogged(t *testing.T) {
	t.Parallel()

	// arrange
	ctx, out := logInto(t)
	clk, setNow := clock.NewMockClock(start)
	handler := requestlog.Interceptor(clk).WrapUnary(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		setNow(start.Add(1500 * time.Millisecond))

		return connect.NewResponse(&struct{}{}), nil
	})

	// act
	_, err := handler(ctx, newRequest(&loginRequest{}))

	// assert
	require.NoError(t, err)

	line := decode(t, out)
	assert.Equal(t, "INFO", line["level"])
	assert.Equal(t, "/tidegate.authority.v1.LoginService/Login", line["procedure"])
	assert.Equal(t, "ok", line["code"])
	assert.InDelta(t, float64(1500*time.Millisecond), line["duration"], 0)
	assert.Equal(t, "192.0.2.7:51234", line["peer"])
}

func TestRefusedCallLogsItsCode(t *testing.T) {
	t.Parallel()

	// arrange
	ctx, out := logInto(t)
	handler := requestlog.Interceptor(fixedClock()).WrapUnary(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("text: value is required"))
	})

	// act
	_, _ = handler(ctx, newRequest(&loginRequest{}))

	// assert
	assert.Equal(t, "invalid_argument", decode(t, out)["code"])
}

func TestExceededDeadlineLogsTheCodeTheClientGets(t *testing.T) {
	t.Parallel()

	// arrange
	ctx, out := logInto(t)
	handler := requestlog.Interceptor(fixedClock()).WrapUnary(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, fmt.Errorf("storing the text: %w", context.DeadlineExceeded)
	})

	// act
	_, _ = handler(ctx, newRequest(&loginRequest{}))

	// assert
	assert.Equal(t, "deadline_exceeded", decode(t, out)["code"])
}

func TestCanceledCallLogsTheCodeTheClientGets(t *testing.T) {
	t.Parallel()

	// arrange
	ctx, out := logInto(t)
	handler := requestlog.Interceptor(fixedClock()).WrapUnary(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, fmt.Errorf("storing the text: %w", context.Canceled)
	})

	// act
	_, _ = handler(ctx, newRequest(&loginRequest{}))

	// assert
	assert.Equal(t, "canceled", decode(t, out)["code"])
}

func TestConnectErrorAroundAContextErrorLogsItsOwnCode(t *testing.T) {
	t.Parallel()

	// arrange
	ctx, out := logInto(t)
	handler := requestlog.Interceptor(fixedClock()).WrapUnary(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, connect.NewError(connect.CodeUnavailable, context.Canceled)
	})

	// act
	_, _ = handler(ctx, newRequest(&loginRequest{}))

	// assert
	assert.Equal(t, "unavailable", decode(t, out)["code"])
}

func TestErrorReachesTheCallerUnchanged(t *testing.T) {
	t.Parallel()

	// arrange
	notFound := connect.NewError(connect.CodeNotFound, errors.New("no text with ID 7"))
	handler := requestlog.Interceptor(fixedClock()).WrapUnary(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, notFound
	})

	// act
	_, err := handler(t.Context(), newRequest(&loginRequest{}))

	// assert
	assert.Same(t, notFound, err)
}

func TestRequestBodyIsNotLogged(t *testing.T) {
	t.Parallel()

	// arrange
	ctx, out := logInto(t)
	handler := requestlog.Interceptor(fixedClock()).WrapUnary(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("wrong user name or password"))
	})

	// act
	_, _ = handler(ctx, newRequest(&loginRequest{Password: "correct horse battery staple"}))

	// assert
	assert.NotContains(t, out.String(), "correct horse battery staple")
}

func TestStreamingCallIsLogged(t *testing.T) {
	t.Parallel()

	// arrange
	ctx, out := logInto(t)
	clk, setNow := clock.NewMockClock(start)
	handler := requestlog.Interceptor(clk).WrapStreamingHandler(func(context.Context, connect.StreamingHandlerConn) error {
		setNow(start.Add(2 * time.Second))

		return connect.NewError(connect.CodeNotFound, errors.New("no session with ID 7"))
	})

	// act
	_ = handler(ctx, stream{})

	// assert
	line := decode(t, out)
	assert.Equal(t, "/tidegate.authority.v1.SessionService/Watch", line["procedure"])
	assert.Equal(t, "not_found", line["code"])
	assert.InDelta(t, float64(2*time.Second), line["duration"], 0)
	assert.Equal(t, "192.0.2.7:51234", line["peer"])
}

func TestStreamingErrorReachesTheCallerUnchanged(t *testing.T) {
	t.Parallel()

	// arrange
	notFound := connect.NewError(connect.CodeNotFound, errors.New("no session with ID 7"))
	handler := requestlog.Interceptor(fixedClock()).WrapStreamingHandler(func(context.Context, connect.StreamingHandlerConn) error {
		return notFound
	})

	// act
	err := handler(t.Context(), stream{})

	// assert
	assert.Same(t, notFound, err)
}

type loginRequest struct {
	Password string
}

// request sets the procedure and peer that connect-go fills in for a real
// call. connect.NewRequest leaves them empty.
type request struct {
	connect.AnyRequest
}

func (request) Spec() connect.Spec {
	return connect.Spec{Procedure: "/tidegate.authority.v1.LoginService/Login"}
}

func (request) Peer() connect.Peer {
	return connect.Peer{Addr: "192.0.2.7:51234"}
}

// stream sets the procedure and peer that connect-go fills in for a real
// call. The interceptor reads nothing else from it.
type stream struct {
	connect.StreamingHandlerConn
}

func (stream) Spec() connect.Spec {
	return connect.Spec{Procedure: "/tidegate.authority.v1.SessionService/Watch"}
}

func (stream) Peer() connect.Peer {
	return connect.Peer{Addr: "192.0.2.7:51234"}
}

func newRequest(message *loginRequest) request {
	return request{AnyRequest: connect.NewRequest(message)}
}

func fixedClock() clock.Service {
	clk, _ := clock.NewMockClock(start)

	return clk
}

func logInto(t *testing.T) (context.Context, *bytes.Buffer) {
	t.Helper()

	var out bytes.Buffer

	return logging.NewContext(t.Context(), logging.New(&out, slog.LevelInfo, logging.FormatJSON)), &out
}

func decode(t *testing.T, out *bytes.Buffer) map[string]any {
	t.Helper()

	var line map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &line))

	return line
}
