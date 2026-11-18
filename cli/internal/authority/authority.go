// Package authority connects the CLI to a tidegate authority.
package authority

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"connectrpc.com/connect"

	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
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
// empty. A ping never gets the token, because the authority answers it
// without a login.
func Connect(server string, token secret.String) (*Clients, error) {
	err := CheckServer(server)
	if err != nil {
		return nil, err
	}

	timeout := timeoutInterceptor{limit: requestTimeout}
	withoutToken := connect.WithInterceptors(timeout)
	withToken := withoutToken

	if token.Reveal() != "" {
		withToken = connect.WithInterceptors(timeout, sessionTokenInterceptor{token: token})
	}

	return &Clients{
		ping:  authorityv1connect.NewPingServiceClient(httpClient, server, withoutToken),
		login: authorityv1connect.NewLoginServiceClient(httpClient, server, withToken),
		dummy: authorityv1connect.NewDummyServiceClient(httpClient, server, withToken),
	}, nil
}

// Ping returns the client for the PingService.
func (c *Clients) Ping() authorityv1connect.PingServiceClient { return c.ping }

// Dummy returns the client for the DummyService.
func (c *Clients) Dummy() authorityv1connect.DummyServiceClient { return c.dummy }

// SignIn checks the user name and the password at the authority at server and
// returns the token of a new session.
func SignIn(ctx context.Context, server, userName string, password secret.String) (secret.String, error) {
	clients, err := Connect(server, secret.String{})
	if err != nil {
		return secret.String{}, err
	}

	response, err := clients.login.Login(ctx, connect.NewRequest(&authorityv1.LoginRequest{
		UserName: userName,
		Password: password.Reveal(),
	}))
	if err != nil {
		return secret.String{}, err
	}

	return secret.New(response.Msg.GetToken()), nil
}

// SignOut ends the session that token belongs to at the authority at server.
func SignOut(ctx context.Context, server string, token secret.String) error {
	clients, err := Connect(server, token)
	if err != nil {
		return err
	}

	_, err = clients.login.Logout(ctx, connect.NewRequest(&authorityv1.LogoutRequest{}))

	return err
}

// CheckServer returns an error unless server is a URL that tidegate sends a
// password or a token to: https, or http to this machine.
func CheckServer(server string) error {
	parsed, err := url.Parse(server)
	if err != nil {
		return errors.New("invalid server URL, want one like https://authority.example")
	}

	if parsed.User != nil {
		return errors.New("the server URL must not contain a user name or password")
	}

	if parsed.Host == "" {
		return errors.New("invalid server URL, want one like https://authority.example")
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
		return fmt.Errorf("invalid server URL %q, want https:// or http:// to localhost", server)
	}
}
