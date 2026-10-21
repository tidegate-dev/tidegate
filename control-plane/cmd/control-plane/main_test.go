package main

import (
	"context"
	"net"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	controlplanev1 "github.com/tidegate-dev/tidegate/gen/tidegate/controlplane/v1"
	"github.com/tidegate-dev/tidegate/gen/tidegate/controlplane/v1/controlplanev1connect"
)

func TestServeAnswersPingOverEveryProtocol(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		protocols func(*http.Protocols)
		options   []connect.ClientOption
	}{
		"connect over HTTP/1.1": {
			protocols: func(p *http.Protocols) { p.SetHTTP1(true) },
		},
		"grpc over unencrypted HTTP/2": {
			protocols: func(p *http.Protocols) { p.SetUnencryptedHTTP2(true) },
			options:   []connect.ClientOption{connect.WithGRPC()},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// arrange
			address := startServer(t)

			var protocols http.Protocols
			test.protocols(&protocols)

			httpClient := &http.Client{Transport: &http.Transport{Protocols: &protocols}}
			client := controlplanev1connect.NewPingServiceClient(httpClient, "http://"+address, test.options...)

			// act
			response, err := client.Ping(t.Context(), connect.NewRequest(&controlplanev1.PingRequest{}))

			// assert
			require.NoError(t, err)
			assert.Equal(t, "pong", response.Msg.GetMessage())
		})
	}
}

func TestServeStopsWhenContextIsDone(t *testing.T) {
	t.Parallel()

	// arrange
	listener := listen(t)
	ctx, cancel := context.WithCancel(t.Context())

	served := make(chan error, 1)

	go func() {
		served <- serve(ctx, listener)
	}()

	// act
	cancel()

	// assert
	require.NoError(t, <-served)
}

func startServer(t *testing.T) string {
	t.Helper()

	listener := listen(t)
	ctx, cancel := context.WithCancel(context.Background())

	served := make(chan error, 1)

	go func() {
		served <- serve(ctx, listener)
	}()

	t.Cleanup(func() {
		cancel()
		assert.NoError(t, <-served)
	})

	return listener.Addr().String()
}

func listen(t *testing.T) net.Listener {
	t.Helper()

	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	return listener
}
