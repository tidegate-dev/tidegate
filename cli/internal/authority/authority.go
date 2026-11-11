// Package authority connects the CLI to a tidegate authority.
package authority

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"connectrpc.com/connect"

	"github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1/authorityv1connect"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

// Clients talks to one authority.
type Clients struct {
	ping  authorityv1connect.PingServiceClient
	login authorityv1connect.LoginServiceClient
	dummy authorityv1connect.DummyServiceClient
}

// Connect refuses a plain HTTP server that isn't on this machine, and returns
// clients that send token with every call, if it isn't empty.
func Connect(server string, token secret.String) (*Clients, error) {
	err := checkServer(server)
	if err != nil {
		return nil, err
	}

	var options []connect.ClientOption
	if token.Reveal() != "" {
		options = append(options, connect.WithInterceptors(bearer(token)))
	}

	return &Clients{
		ping:  authorityv1connect.NewPingServiceClient(http.DefaultClient, server, options...),
		login: authorityv1connect.NewLoginServiceClient(http.DefaultClient, server, options...),
		dummy: authorityv1connect.NewDummyServiceClient(http.DefaultClient, server, options...),
	}, nil
}

// Ping returns the client for the PingService.
func (c *Clients) Ping() authorityv1connect.PingServiceClient { return c.ping }

// Login returns the client for the LoginService.
func (c *Clients) Login() authorityv1connect.LoginServiceClient { return c.login }

// Dummy returns the client for the DummyService.
func (c *Clients) Dummy() authorityv1connect.DummyServiceClient { return c.dummy }

func checkServer(server string) error {
	parsed, err := url.Parse(server)
	if err != nil {
		return fmt.Errorf("invalid --server %q: %w", server, err)
	}

	if parsed.Scheme != "http" {
		return nil
	}

	switch parsed.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return nil
	default:
		return fmt.Errorf("refusing to send a password or a token over plain HTTP to %s, use https:// or a server on localhost", server)
	}
}

func bearer(token secret.String) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
			request.Header().Set("Authorization", "Bearer "+token.Reveal())

			return next(ctx, request)
		}
	}
}
