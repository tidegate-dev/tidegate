package model

import (
	"context"
	"errors"
)

// ErrDummyNotFound means no dummy has the given ID.
var ErrDummyNotFound = errors.New("dummy not found")

// Dummy is a text that checks the path from a client into the database and
// back.
type Dummy struct {
	// ID is 0 until the dummy is stored.
	ID   int64
	Text string
}

// DummyReadWriter belongs to a Unit. It reads dummies right away and records
// writes, which the unit runs at Commit.
type DummyReadWriter interface {
	// ByID returns the dummy with the ID, or ErrDummyNotFound.
	ByID(ctx context.Context, id int64) (*Dummy, error)
	// Add records the insert of dummy. Commit gives dummy its ID.
	Add(dummy *Dummy)
}
