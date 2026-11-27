package command

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"

	"github.com/phemera-dev/phemera/cli/internal/connections"
	authorityv1 "github.com/phemera-dev/phemera/gen/phemera/authority/v1"
	"github.com/phemera-dev/phemera/gen/phemera/authority/v1/authorityv1connect"
)

// fakeTicketService stands in for an authority. It issues a ticket for every
// asset.
type fakeTicketService struct {
	authorityv1connect.UnimplementedTicketServiceHandler
}

func (fakeTicketService) CreateTicket(
	context.Context, *connect.Request[authorityv1.CreateTicketRequest],
) (*connect.Response[authorityv1.CreateTicketResponse], error) {
	return connect.NewResponse(&authorityv1.CreateTicketResponse{}), nil
}

// storeOfOneConnection stands in for the saved connections. Every command
// uses its connection.
type storeOfOneConnection struct {
	connections.Store

	connection connections.Connection
}

func (store storeOfOneConnection) Current(string) (connections.Connection, error) {
	return store.connection, nil
}

// connectThroughSSHProxyAt runs phemera connect against an authority that
// issues the ticket, with the ssh-proxy expected at sshProxyAddress.
func connectThroughSSHProxyAt(t *testing.T, sshProxyAddress string) error {
	t.Helper()

	handlers := http.NewServeMux()
	handlers.Handle(authorityv1connect.NewTicketServiceHandler(fakeTicketService{}))

	authority := httptest.NewServer(handlers)
	t.Cleanup(authority.Close)

	store := storeOfOneConnection{connection: connections.Connection{Name: "first", Server: authority.URL}}

	phemera := &cli.Command{
		Name:      "phemera",
		Commands:  []*cli.Command{connectCommand(store, sshProxyAddress)},
		Reader:    strings.NewReader(""),
		Writer:    io.Discard,
		ErrWriter: io.Discard,
	}

	return phemera.Run(t.Context(), []string{"phemera", "connect", "00000000-0000-4000-8000-000000000000"})
}

func TestConnectExitsWithStatus255WhenItCannotReachTheSSHProxy(t *testing.T) {
	t.Parallel()

	// arrange
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	address := listener.Addr().String()
	require.NoError(t, listener.Close())

	// act
	err = connectThroughSSHProxyAt(t, address)

	// assert
	require.ErrorContains(t, err, "connecting to the ssh-proxy")
	assert.Equal(t, 255, ExitStatus(err))
}
