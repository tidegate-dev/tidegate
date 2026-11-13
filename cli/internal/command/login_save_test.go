package command

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/cli/internal/connections"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1/authorityv1connect"
)

// fakeLoginService stands in for an authority. It gives out the session token
// "new token" and records which sessions are ended.
type fakeLoginService struct {
	authorityv1connect.UnimplementedLoginServiceHandler

	refuseToEndSessions bool

	mutex               sync.Mutex
	endedAuthorizations []string
}

func (service *fakeLoginService) Login(
	context.Context, *connect.Request[authorityv1.LoginRequest],
) (*connect.Response[authorityv1.LoginResponse], error) {
	return connect.NewResponse(&authorityv1.LoginResponse{Token: "new token"}), nil
}

func (service *fakeLoginService) Logout(
	_ context.Context, request *connect.Request[authorityv1.LogoutRequest],
) (*connect.Response[authorityv1.LogoutResponse], error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	if service.refuseToEndSessions {
		return nil, connect.NewError(connect.CodeInternal, errors.New("the database is down"))
	}

	service.endedAuthorizations = append(service.endedAuthorizations, request.Header().Get("Authorization"))

	return connect.NewResponse(&authorityv1.LogoutResponse{}), nil
}

func (service *fakeLoginService) authorizationsOfEndedSessions() []string {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	return service.endedAuthorizations
}

// storeThatCannotSave stands in for the saved connections. It names a new
// connection after its URL and fails every Save.
type storeThatCannotSave struct {
	connections.Store
}

func (storeThatCannotSave) ForLogin(url, _ string) (connections.Connection, error) {
	return connections.Connection{Name: url, Server: url}, nil
}

func (storeThatCannotSave) Save(connections.Connection) (connections.Connection, error) {
	return connections.Connection{}, errors.New("the disk is full")
}

// loginWhenSavingFails runs tidegate login against service, with saved
// connections that can't be written.
func loginWhenSavingFails(t *testing.T, service *fakeLoginService) error {
	t.Helper()

	handlers := http.NewServeMux()
	handlers.Handle(authorityv1connect.NewLoginServiceHandler(service))

	authority := httptest.NewServer(handlers)
	t.Cleanup(authority.Close)

	tidegate := New(storeThatCannotSave{})
	tidegate.Reader = strings.NewReader("alice\nhunter2\n")
	tidegate.Writer = io.Discard
	tidegate.ErrWriter = io.Discard

	return tidegate.Run(t.Context(), []string{"tidegate", "login", authority.URL})
}

func TestLoginEndsTheNewSessionWhenItCannotSaveIt(t *testing.T) {
	t.Parallel()

	// arrange
	service := &fakeLoginService{}

	// act
	err := loginWhenSavingFails(t, service)

	// assert
	require.Error(t, err)
	assert.Equal(t, []string{"Bearer new token"}, service.authorizationsOfEndedSessions())
}

func TestLoginSaysWhenItCanNeitherSaveNorEndTheNewSession(t *testing.T) {
	t.Parallel()

	// arrange
	service := &fakeLoginService{refuseToEndSessions: true}

	// act
	err := loginWhenSavingFails(t, service)

	// assert
	require.Error(t, err)
	assert.Contains(t, Message(err), "stays valid until it expires")
}
