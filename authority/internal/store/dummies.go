package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
	"github.com/tidegate-dev/tidegate/authority/internal/model"
)

// DummyReadWriter queries the dummies table on the connection pool. It
// writes right away, without a unit of work.
type DummyReadWriter struct {
	queries *sqlc.Queries
}

// Add stores text and returns its ID.
func (d *DummyReadWriter) Add(ctx context.Context, text string) (int64, error) {
	id, err := d.queries.CreateDummy(ctx, text)
	if err != nil {
		return 0, fmt.Errorf("storing the text: %w", err)
	}

	return id, nil
}

// ByID returns the text with the ID, or model.ErrDummyNotFound.
func (d *DummyReadWriter) ByID(ctx context.Context, id int64) (string, error) {
	text, err := d.queries.GetDummy(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", model.ErrDummyNotFound
	}

	if err != nil {
		return "", fmt.Errorf("reading the text: %w", err)
	}

	return text, nil
}
