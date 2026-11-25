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
	login   authorityv1connect.LoginServiceClient
	project authorityv1connect.ProjectServiceClient
	role    authorityv1connect.RoleServiceClient
	asset   authorityv1connect.AssetServiceClient
	binding authorityv1connect.BindingServiceClient
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
	err := CheckServer(server)
	if err != nil {
		return nil, err
	}

	interceptors := []connect.Interceptor{timeoutInterceptor{limit: requestTimeout}}
	if token.Reveal() != "" {
		interceptors = append(interceptors, sessionTokenInterceptor{token: token})
	}

	options := connect.WithInterceptors(interceptors...)

	return &Clients{
		login:   authorityv1connect.NewLoginServiceClient(httpClient, server, options),
		project: authorityv1connect.NewProjectServiceClient(httpClient, server, options),
		role:    authorityv1connect.NewRoleServiceClient(httpClient, server, options),
		asset:   authorityv1connect.NewAssetServiceClient(httpClient, server, options),
		binding: authorityv1connect.NewBindingServiceClient(httpClient, server, options),
	}, nil
}

// Project returns the client for the ProjectService.
func (c *Clients) Project() authorityv1connect.ProjectServiceClient { return c.project }

// Role returns the client for the RoleService.
func (c *Clients) Role() authorityv1connect.RoleServiceClient { return c.role }

// Asset returns the client for the AssetService.
func (c *Clients) Asset() authorityv1connect.AssetServiceClient { return c.asset }

// Binding returns the client for the BindingService.
func (c *Clients) Binding() authorityv1connect.BindingServiceClient { return c.binding }

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
