package service

import (
	"context"
	"errors"

	"github.com/phemera-dev/phemera/authority/internal/model"
)

// ErrAssetAccessDenied means the user may not access the requested asset.
var ErrAssetAccessDenied = errors.New("access to the asset is denied")

// Tickets issues tickets for asset access requests.
type Tickets struct {
	units model.UnitFactory
}

// NewTickets returns Tickets that write in a unit of work.
func NewTickets(units model.UnitFactory) *Tickets {
	return &Tickets{units: units}
}

// Create records the authenticated user's request and returns the committed ticket.
func (t *Tickets) Create(ctx context.Context, userID model.UserID, assetID model.AssetID) (*model.Ticket, error) {
	if !bindingsAllowAssetAccess(ctx, userID, assetID) {
		return nil, ErrAssetAccessDenied
	}

	id, err := model.NewTicketID()
	if err != nil {
		return nil, err
	}

	ticket := &model.Ticket{ID: id, AssetID: assetID, UserID: userID}
	unit := t.units.NewUnit()
	unit.Tickets().Add(ticket)

	err = unit.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return ticket, nil
}

// bindingsAllowAssetAccess currently approves every request.
// TODO: Find the user's bindings in the asset's project and check whether
// the permissions of a bound role allow access to the requested asset.
func bindingsAllowAssetAccess(_ context.Context, _ model.UserID, _ model.AssetID) bool {
	return true
}
