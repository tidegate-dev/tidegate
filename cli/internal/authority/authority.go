// Package authority connects the CLI to a tidegate authority.
package authority

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

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

const requestTimeout = 10 * time.Second

var httpClient = &http.Client{
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// Connect refuses a plain HTTP server that isn't on this machine, and returns
// clients that give every call 10 seconds and send token with it, if it isn't
// empty.
func Connect(server string, token secret.String) (*Clients, error) {
	err := checkServer(server)
	if err != nil {
		return nil, err
	}

	interceptors := []connect.Interceptor{timeoutInterceptor{limit: requestTimeout}}
	if token.Reveal() != "" {
		interceptors = append(interceptors, sessionTokenInterceptor{token: token})
	}

	option := connect.WithInterceptors(interceptors...)

	return &Clients{
		ping:  authorityv1connect.NewPingServiceClient(httpClient, server, option),
		login: authorityv1connect.NewLoginServiceClient(httpClient, server, option),
		dummy: authorityv1connect.NewDummyServiceClient(httpClient, server, option),
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

	if parsed.User != nil {
		return errors.New("--server must not contain a user name or password")
	}

	if parsed.Host == "" {
		return fmt.Errorf("invalid --server %q, want a URL like https://authority.example", server)
	}

	switch parsed.Scheme {
	case "https":
		return nil
	case "http":
		switch parsed.Hostname() {
		case "localhost", "127.0.0.1", "::1":
			return nil
		default:
			return fmt.Errorf("refusing to send a password or a token over plain HTTP to %s, use https:// or a server on localhost", server)
		}
	default:
		return fmt.Errorf("invalid --server %q, want https:// or http:// to localhost", server)
	}
}
