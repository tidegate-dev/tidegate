package api

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

// TicketService records asset access requests.
type TicketService struct {
	tickets *service.Tickets
}

// NewTicketService returns a TicketService.
func NewTicketService(tickets *service.Tickets) *TicketService {
	return &TicketService{tickets: tickets}
}

// CreateTicket issues a ticket to the authenticated user for the asset.
func (s *TicketService) CreateTicket(
	ctx context.Context,
	req *connect.Request[authorityv1.CreateTicketRequest],
) (*connect.Response[authorityv1.CreateTicketResponse], error) {
	session, err := sessionFromContext(ctx)
	if err != nil {
		return nil, clientError(service.ErrNotSignedIn)
	}

	id, err := uuid.Parse(req.Msg.GetAssetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("asset_id must be a UUID"))
	}

	ticket, err := s.tickets.Create(ctx, session.UserID, model.AssetID(id))
	if err != nil {
		return nil, clientError(err)
	}

	return connect.NewResponse(&authorityv1.CreateTicketResponse{
		Id:      ticket.ID.String(),
		AssetId: ticket.AssetID.String(),
		UserId:  ticket.UserID.String(),
	}), nil
}
