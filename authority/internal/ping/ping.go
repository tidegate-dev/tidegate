// Package ping implements the PingService, which lets a client check that
// it can reach the authority.
package ping

import (
	"context"

	"connectrpc.com/connect"

	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

// Service answers pings.
type Service struct{}

// Ping answers with pong.
func (Service) Ping(
	context.Context,
	*connect.Request[authorityv1.PingRequest],
) (*connect.Response[authorityv1.PingResponse], error) {
	return connect.NewResponse(&authorityv1.PingResponse{Message: "pong"}), nil
}
