package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
)

// ErrNotFound means the database holds nothing with the given ID.
var ErrNotFound = errors.New("not found")

// Dummies stores texts and reads them back, to check the path from a client
// through the authority into the database.
type Dummies struct {
	queries *sqlc.Queries
}

// NewDummies returns Dummies that stores texts in pool.
func NewDummies(pool *pgxpool.Pool) *Dummies {
	return &Dummies{queries: sqlc.New(pool)}
}

// Write stores text and returns its ID.
func (d *Dummies) Write(ctx context.Context, text string) (int64, error) {
	id, err := d.queries.CreateDummy(ctx, text)
	if err != nil {
		return 0, fmt.Errorf("storing the text: %w", err)
	}

	return id, nil
}

// Read returns the text with the given ID, or ErrNotFound.
func (d *Dummies) Read(ctx context.Context, id int64) (string, error) {
	text, err := d.queries.GetDummy(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}

	if err != nil {
		return "", fmt.Errorf("reading the text: %w", err)
	}

	return text, nil
}
