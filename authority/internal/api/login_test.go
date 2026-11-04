package api

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"

	"github.com/tidegate-dev/tidegate/authority/internal/password"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
)

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
