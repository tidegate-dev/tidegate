package api

import (
	"context"

	"connectrpc.com/connect"

	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

// PingService answers pings, so a client can check that it reaches the
// authority.
type PingService struct{}

// Ping answers with pong.
func (PingService) Ping(
	context.Context,
	*connect.Request[authorityv1.PingRequest],
) (*connect.Response[authorityv1.PingResponse], error) {
	return connect.NewResponse(&authorityv1.PingResponse{Message: "pong"}), nil
}
