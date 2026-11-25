package store

import (
	"context"
	"fmt"

	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
	"github.com/tidegate-dev/tidegate/authority/internal/model"
)

// TicketReadWriter records ticket writes, which the unit runs at Commit.
type TicketReadWriter struct {
	unit *Unit
}

// Add records the insert of ticket.
func (t *TicketReadWriter) Add(ticket *model.Ticket) {
	t.unit.add(func(ctx context.Context, queries *sqlc.Queries) error {
		version, err := queries.InsertTicket(ctx, sqlc.InsertTicketParams{
			ID:      databaseID(ticket.ID),
			AssetID: databaseID(ticket.AssetID),
			UserID:  databaseID(ticket.UserID),
		})
		if err != nil {
			return fmt.Errorf("creating the ticket: %w", err)
		}

		ticket.Version = version.Uint32

		return nil
	})
}
