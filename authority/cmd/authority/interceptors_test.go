package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/The127/go-clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/tidegate-dev/tidegate/internal/logging"
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
		connect.WithInterceptors(interceptors(clock.NewSystemClock())...)))

	server := httptest.NewUnstartedServer(mux)
	server.Config.BaseContext = func(net.Listener) context.Context {
		return logging.NewContext(context.Background(), logging.New(&out, slog.LevelInfo, logging.FormatText))
	}
	server.Start()
	t.Cleanup(server.Close)

	client := connect.NewClient[emptypb.Empty, emptypb.Empty](server.Client(), server.URL+procedure)

	// act
	_, err := client.CallUnary(t.Context(), connect.NewRequest(&emptypb.Empty{}))

	// assert
	require.Error(t, err)
	assert.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	assert.Contains(t, out.String(), "code=internal")
}
