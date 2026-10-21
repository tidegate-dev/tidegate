// Package ping implements the PingService, which lets a client check that
// it can reach the control plane.
package ping

import (
	"context"

	"connectrpc.com/connect"

	controlplanev1 "github.com/tidegate-dev/tidegate/gen/tidegate/controlplane/v1"
)

// Service answers pings.
type Service struct{}

// Ping answers with pong.
func (Service) Ping(
	context.Context,
	*connect.Request[controlplanev1.PingRequest],
) (*connect.Response[controlplanev1.PingResponse], error) {
	return connect.NewResponse(&controlplanev1.PingResponse{Message: "pong"}), nil
}
