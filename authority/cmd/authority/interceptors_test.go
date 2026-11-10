package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/The127/go-clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/tidegate-dev/tidegate/authority/internal/service"
	"github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1/authorityv1connect"
	"github.com/tidegate-dev/tidegate/internal/logging"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

func TestRequestLogShowsTheCodeTheClientGets(t *testing.T) {
	t.Parallel()

	// arrange
	var out bytes.Buffer

	const procedure = "/tidegate.test.v1.TestService/Fail"

	mux := http.NewServeMux()
	mux.Handle(procedure, connect.NewUnaryHandler(procedure,
		func(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
			return nil, errors.New("reading the text: connection refused")
		},
		connect.WithInterceptors(interceptors(clock.NewSystemClock(), anySession{})...)))

	server := httptest.NewUnstartedServer(mux)
	server.Config.BaseContext = func(net.Listener) context.Context {
		return logging.NewContext(context.Background(), logging.New(&out, slog.LevelInfo, logging.FormatText))
	}
	server.Start()

	closeServerOnce := sync.OnceFunc(server.Close)
	t.Cleanup(closeServerOnce)

	client := connect.NewClient[emptypb.Empty, emptypb.Empty](server.Client(), server.URL+procedure)
	request := connect.NewRequest(&emptypb.Empty{})
	request.Header().Set("Authorization", "Bearer abc")

	// act
	_, err := client.CallUnary(t.Context(), request)

	closeServerOnce()

	// assert
	require.Error(t, err)
	assert.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	assert.Contains(t, out.String(), "code=internal")
}

func TestInterceptorsRefuseACallWithoutALoginToken(t *testing.T) {
	t.Parallel()

	// arrange
	client := clientOfSucceedingHandler(t, "/tidegate.test.v1.TestService/Succeed")

	// act
	_, err := client.CallUnary(t.Context(), connect.NewRequest(&emptypb.Empty{}))

	// assert
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestInterceptorsLetLoginThroughWithoutALoginToken(t *testing.T) {
	t.Parallel()

	// arrange
	client := clientOfSucceedingHandler(t, authorityv1connect.LoginServiceLoginProcedure)

	// act
	_, err := client.CallUnary(t.Context(), connect.NewRequest(&emptypb.Empty{}))

	// assert
	assert.NoError(t, err)
}

func clientOfSucceedingHandler(t *testing.T, procedure string) *connect.Client[emptypb.Empty, emptypb.Empty] {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(procedure, connect.NewUnaryHandler(procedure,
		func(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
			return connect.NewResponse(&emptypb.Empty{}), nil
		},
		connect.WithInterceptors(interceptors(clock.NewSystemClock(), anySession{})...)))

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return connect.NewClient[emptypb.Empty, emptypb.Empty](server.Client(), server.URL+procedure)
}

type anySession struct{}

func (anySession) Session(context.Context, secret.String) (service.Session, error) {
	return service.Session{}, nil
}
