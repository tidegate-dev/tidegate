package ping_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/authority/internal/ping"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1/authorityv1connect"
)

func TestPingAnswersPong(t *testing.T) {
	t.Parallel()

	// arrange
	mux := http.NewServeMux()
	mux.Handle(authorityv1connect.NewPingServiceHandler(ping.Service{}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := authorityv1connect.NewPingServiceClient(server.Client(), server.URL)

	// act
	response, err := client.Ping(t.Context(), connect.NewRequest(&authorityv1.PingRequest{}))

	// assert
	require.NoError(t, err)
	assert.Equal(t, "pong", response.Msg.GetMessage())
}
