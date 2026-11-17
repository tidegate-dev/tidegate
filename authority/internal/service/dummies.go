package service

import "context"

// DummyStore stores texts.
type DummyStore interface {
	// Add stores text and returns its ID.
	Add(ctx context.Context, text string) (int64, error)
	// ByID returns the text with the ID, or model.ErrDummyNotFound.
	ByID(ctx context.Context, id int64) (string, error)
}

// Dummies checks the path from a client into the database and back.
type Dummies struct {
	store DummyStore
}

// NewDummies returns Dummies that keep their texts in store.
func NewDummies(store DummyStore) *Dummies {
	return &Dummies{store: store}
}

// Write stores text and returns its ID.
func (d *Dummies) Write(ctx context.Context, text string) (int64, error) {
	return d.store.Add(ctx, text)
}

// Read returns the text with the given ID, or model.ErrDummyNotFound.
func (d *Dummies) Read(ctx context.Context, id int64) (string, error) {
	return d.store.ByID(ctx, id)
}
