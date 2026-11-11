package authority

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

func TestConnectDoesntFollowRedirects(t *testing.T) {
	t.Parallel()

	// arrange
	var reached atomic.Bool

	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		reached.Store(true)
	}))
	defer target.Close()

	redirecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer redirecting.Close()

	clients, err := Connect(redirecting.URL, secret.New("token"))
	require.NoError(t, err)

	// act
	_, err = clients.Ping().Ping(t.Context(), connect.NewRequest(&authorityv1.PingRequest{}))

	// assert
	require.Error(t, err)
	assert.False(t, reached.Load())
}

func TestCheckServerAcceptsHTTPSAndLocalHTTP(t *testing.T) {
	t.Parallel()

	for _, server := range []string{
		"https://authority.example",
		"https://authority.example:8443/prefix",
		"HTTPS://authority.example",
		"http://localhost:8080",
		"http://127.0.0.1:8080",
		"http://[::1]:8080",
	} {
		t.Run(server, func(t *testing.T) {
			t.Parallel()

			// act
			err := checkServer(server)

			// assert
			assert.NoError(t, err)
		})
	}
}

func TestCheckServerRefusesOtherServers(t *testing.T) {
	t.Parallel()

	for _, server := range []string{
		"http://authority.example",
		"http://10.0.0.5:8080",
		"localhost:8080",
		"authority.example",
		"ftp://authority.example",
		"//authority.example",
		"https://",
		"",
	} {
		t.Run(server, func(t *testing.T) {
			t.Parallel()

			// act
			err := checkServer(server)

			// assert
			assert.Error(t, err)
		})
	}
}
