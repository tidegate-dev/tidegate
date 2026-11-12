package command

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

// fakeLoginService gives out one session token and records which tokens are signed out.
type fakeLoginService struct {
	authorityv1connect.UnimplementedLoginServiceHandler

	mutex      sync.Mutex
	signedOut  []string
	failLogout bool
}

func (f *fakeLoginService) Login(context.Context, *connect.Request[authorityv1.LoginRequest]) (*connect.Response[authorityv1.LoginResponse], error) {
	return connect.NewResponse(&authorityv1.LoginResponse{Token: "new token"}), nil
}

func (f *fakeLoginService) Logout(_ context.Context, request *connect.Request[authorityv1.LogoutRequest]) (*connect.Response[authorityv1.LogoutResponse], error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.failLogout {
		return nil, connect.NewError(connect.CodeInternal, errors.New("the database is down"))
	}

	f.signedOut = append(f.signedOut, request.Header().Get("Authorization"))

	return connect.NewResponse(&authorityv1.LogoutResponse{}), nil
}

func (f *fakeLoginService) authorizationsSignedOut() []string {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	return f.signedOut
}

// loginWithoutASaveableFile runs tidegate login against service, with a
// directory for the saved connections that can't be written to.
func loginWithoutASaveableFile(t *testing.T, service *fakeLoginService) error {
	t.Helper()

	if os.Geteuid() == 0 {
		t.Skip("root can write to a read-only directory")
	}

	mux := http.NewServeMux()
	mux.Handle(authorityv1connect.NewLoginServiceHandler(service))

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("TIDEGATE_CONNECTION", "")

	path, err := connections.DefaultPath()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.Chmod(filepath.Dir(path), 0o500)) //nolint:gosec // A directory needs the execute bit.

	root := New()
	root.Reader = strings.NewReader("alice\nhunter2\n")
	root.Writer = io.Discard
	root.ErrWriter = io.Discard

	return root.Run(t.Context(), []string{"tidegate", "login", server.URL})
}

func TestLoginEndsTheNewSessionWhenItCannotSaveIt(t *testing.T) {
	// arrange
	service := &fakeLoginService{}

	// act
	err := loginWithoutASaveableFile(t, service)

	// assert
	require.Error(t, err)
	assert.Equal(t, []string{"Bearer new token"}, service.authorizationsSignedOut())
}

func TestLoginSaysWhenItCanNeitherSaveNorEndTheNewSession(t *testing.T) {
	// arrange
	service := &fakeLoginService{failLogout: true}

	// act
	err := loginWithoutASaveableFile(t, service)

	// assert
	require.Error(t, err)
	assert.Contains(t, Message(err), "stays valid until it expires")
}
