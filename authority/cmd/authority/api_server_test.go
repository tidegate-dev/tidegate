package main

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/authority/internal/api"
	"github.com/tidegate-dev/tidegate/authority/internal/config"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1/authorityv1connect"
)

func TestAPIServerRefusesALoginRequestOverTheLimit(t *testing.T) {
	t.Parallel()

	// arrange
	// The constructors only keep the pool, and the request never reaches it.
	authenticator, err := service.NewAuthenticator(nil)
	require.NoError(t, err)

	logins := service.NewLogins(authenticator, service.NewLoginTokens(nil, time.Hour))
	server := httptest.NewServer(apiServer(config.Server{}, nil, logins).Handler)
	t.Cleanup(server.Close)

	client := authorityv1connect.NewLoginServiceClient(server.Client(), server.URL)
	request := connect.NewRequest(&authorityv1.LoginRequest{
		UserName: "alice",
		Password: strings.Repeat("a", api.MaxLoginRequestBytes),
	})

	// act
	_, err = client.Login(t.Context(), request)

	// assert
	assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
}
