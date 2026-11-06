package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"

	"github.com/tidegate-dev/tidegate/authority/internal/password"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
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

func TestClientErrorLeavesOutTheReasonOfWrongCredentials(t *testing.T) {
	t.Parallel()

	// act
	err := clientError(fmt.Errorf("%w: unknown user", service.ErrWrongCredentials))

	// assert
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.NotContains(t, err.Error(), "unknown user")
}

func TestClientErrorGivesEachErrorItsCode(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		err  error
		code connect.Code
	}{
		"password too long": {fmt.Errorf("checking: %w", password.ErrTooLong), connect.CodeInvalidArgument},
		"busy":              {service.ErrBusy, connect.CodeUnavailable},
		"canceled":          {context.Canceled, connect.CodeUnknown},
		"anything else":     {errors.New("database down"), connect.CodeUnknown},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			err := clientError(test.err)

			// assert
			assert.Equal(t, test.code, connect.CodeOf(err))
		})
	}
}
