package command

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/cli/internal/connections"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1/authorityv1connect"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

func TestConnectRequestsATicketWithTheCurrentSession(t *testing.T) {
	t.Parallel()

	const (
		assetID  = "c809392f-043b-4a61-91b8-80b6f9a4827d"
		ticketID = "16d9ad71-03ae-4056-ba18-5f7dea9a6b7d"
		userID   = "e091d150-5d15-41ec-b8d3-77bc2dc9ab89"
	)

	mux := http.NewServeMux()
	mux.Handle(authorityv1connect.TicketServiceCreateTicketProcedure,
		connect.NewUnaryHandler(authorityv1connect.TicketServiceCreateTicketProcedure,
			func(_ context.Context, req *connect.Request[authorityv1.CreateTicketRequest]) (*connect.Response[authorityv1.CreateTicketResponse], error) {
				assert.Equal(t, assetID, req.Msg.GetAssetId())
				assert.Equal(t, "Bearer test-token", req.Header().Get("Authorization"))

				return connect.NewResponse(&authorityv1.CreateTicketResponse{
					Id: ticketID, AssetId: assetID, UserId: userID,
				}), nil
			},
		),
	)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	var output bytes.Buffer

	cmd := New(ticketConnection{connection: connections.Connection{
		Server: server.URL,
		Token:  secret.New("test-token"),
	}})
	cmd.Writer = &output

	err := cmd.Run(t.Context(), []string{"tidegate", "--output", "json", "connect", assetID})

	require.NoError(t, err)
	assert.JSONEq(t, `{"id":"`+ticketID+`","assetId":"`+assetID+`","userId":"`+userID+`"}`, output.String())
}

type ticketConnection struct {
	connections.Store

	connection connections.Connection
}

func (c ticketConnection) Current(string) (connections.Connection, error) {
	return c.connection, nil
}
