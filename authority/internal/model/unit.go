package model

import "context"

// Unit is a unit of work. It collects writes and runs them in one transaction
// at Commit. Until then it writes nothing.
type Unit interface {
	// Users returns the users for this unit.
	Users() UserReadWriter
	// Sessions returns the sessions for this unit.
	Sessions() SessionReadWriter
	// Dummies returns the dummies for this unit.
	Dummies() DummyReadWriter
	// Projects returns the projects for this unit.
	Projects() ProjectReadWriter
	// Roles returns the roles for this unit.
	Roles() RoleReadWriter
	// Commit runs the pending writes in the order they were added. If one
	// fails, none is kept. It returns ErrChangedSinceRead if a row changed
	// after it was read. If it fails, the models it wrote may hold versions
	// that were never stored, so read them again before you write them. A
	// unit can be committed once.
	Commit(ctx context.Context) error
}

// UnitFactory starts units of work.
type UnitFactory interface {
	// NewUnit returns an empty unit of work.
	NewUnit() Unit
}
