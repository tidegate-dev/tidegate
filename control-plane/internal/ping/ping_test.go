package ping_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/control-plane/internal/ping"
	controlplanev1 "github.com/tidegate-dev/tidegate/gen/tidegate/controlplane/v1"
	"github.com/tidegate-dev/tidegate/gen/tidegate/controlplane/v1/controlplanev1connect"
)

func TestPingAnswersPong(t *testing.T) {
	t.Parallel()

	// arrange
	mux := http.NewServeMux()
	mux.Handle(controlplanev1connect.NewPingServiceHandler(ping.Service{}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := controlplanev1connect.NewPingServiceClient(server.Client(), server.URL)

	// act
	response, err := client.Ping(t.Context(), connect.NewRequest(&controlplanev1.PingRequest{}))

	// assert
	require.NoError(t, err)
	assert.Equal(t, "pong", response.Msg.GetMessage())
}
