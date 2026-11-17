package model

import "context"

// Unit is a unit of work. It collects writes and runs them in one transaction
// at Commit. Until then it writes nothing.
type Unit interface {
	// Users returns the users for this unit.
	Users() UserReadWriter
	// Sessions returns the sessions for this unit.
	Sessions() SessionReadWriter
	// Commit runs the pending writes in the order they were added. If one
	// fails, none is kept. It returns ErrConflict if a row changed after it
	// was read. A unit can be committed once.
	Commit(ctx context.Context) error
}

// UnitFactory starts units of work.
type UnitFactory interface {
	// NewUnit returns an empty unit of work.
	NewUnit() Unit
}
