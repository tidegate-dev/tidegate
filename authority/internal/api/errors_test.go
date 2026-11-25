package api

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"

	"github.com/phemera-dev/phemera/authority/internal/model"
	"github.com/phemera-dev/phemera/authority/internal/password"
	"github.com/phemera-dev/phemera/authority/internal/service"
	"github.com/phemera-dev/phemera/authority/internal/sshkey"
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
		"password too long":         {fmt.Errorf("checking: %w", password.ErrTooLong), connect.CodeInvalidArgument},
		"not signed in":             {fmt.Errorf("%w: expired token", service.ErrNotSignedIn), connect.CodeUnauthenticated},
		"busy":                      {service.ErrBusy, connect.CodeUnavailable},
		"project name taken":        {model.ErrProjectNameTaken, connect.CodeAlreadyExists},
		"project not found":         {fmt.Errorf("reading: %w", model.ErrProjectNotFound), connect.CodeNotFound},
		"role name taken":           {model.ErrRoleNameTaken, connect.CodeAlreadyExists},
		"role not found":            {fmt.Errorf("reading: %w", model.ErrRoleNotFound), connect.CodeNotFound},
		"asset name taken":          {model.ErrAssetNameTaken, connect.CodeAlreadyExists},
		"binding exists":            {model.ErrBindingExists, connect.CodeAlreadyExists},
		"empty SSH password":        {service.ErrSSHPasswordEmpty, connect.CodeInvalidArgument},
		"invalid SSH private key":   {sshkey.ErrPrivateKeyInvalid, connect.CodeInvalidArgument},
		"encrypted SSH private key": {sshkey.ErrPrivateKeyEncrypted, connect.CodeInvalidArgument},
		"user not found":            {fmt.Errorf("reading: %w", model.ErrUserNotFound), connect.CodeNotFound},
		"canceled":                  {context.Canceled, connect.CodeUnknown},
		"anything else":             {errors.New("database down"), connect.CodeUnknown},
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
