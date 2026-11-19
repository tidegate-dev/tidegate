package api

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"connectrpc.com/validate"
	"google.golang.org/protobuf/types/known/emptypb"
)

// validationError returns what the interceptor that the server uses answers
// to a request with the message.
func validationError[Message any](t *testing.T, message *Message) error {
	t.Helper()

	handled := func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		return connect.NewResponse(&emptypb.Empty{}), nil
	}

	_, err := validate.NewInterceptor().WrapUnary(handled)(t.Context(), connect.NewRequest(message))

	return err
}
