package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"

	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1/authorityv1connect"
)

func TestLoginServiceHandlerRefusesARequestOverTheLimit(t *testing.T) {
	t.Parallel()

	// arrange
	mux := http.NewServeMux()
	// The handler refuses the request before it reaches logins.
	mux.Handle(NewLoginServiceHandler(nil))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := authorityv1connect.NewLoginServiceClient(server.Client(), server.URL)
	request := connect.NewRequest(&authorityv1.LoginRequest{
		UserName: "alice",
		Password: strings.Repeat("a", maxLoginRequestBytes),
	})

	// act
	_, err := client.Login(t.Context(), request)

	// assert
	assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
}
