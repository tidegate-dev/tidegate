package model

import (
	"fmt"

	"github.com/google/uuid"
)

// TicketID identifies a ticket.
type TicketID uuid.UUID

// NewTicketID returns the ID for a new ticket.
func NewTicketID() (TicketID, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return TicketID{}, fmt.Errorf("creating a ticket ID: %w", err)
	}

	return TicketID(id), nil
}

// String returns the ID in the form xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.
func (id TicketID) String() string {
	return uuid.UUID(id).String()
}

// Ticket records an approved asset access request.
type Ticket struct {
	ID      TicketID
	AssetID AssetID
	UserID  UserID
	// Version holds the row version set by the store.
	Version uint32
}

// TicketReadWriter belongs to a Unit. It records writes that run at Commit.
type TicketReadWriter interface {
	// Add records the insert of ticket.
	Add(ticket *Ticket)
}
