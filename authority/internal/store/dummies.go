package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
	"github.com/tidegate-dev/tidegate/authority/internal/model"
)

// DummyReader queries the dummies table on the connection pool.
type DummyReader struct {
	queries *sqlc.Queries
}

// FindByID returns the dummy with the ID, or model.ErrDummyNotFound.
func (d *DummyReader) FindByID(ctx context.Context, id int64) (*model.Dummy, error) {
	text, err := d.queries.GetDummy(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrDummyNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("reading the dummy: %w", err)
	}

	return &model.Dummy{ID: id, Text: text}, nil
}

// DummyReadWriter belongs to a unit of work. It queries like DummyReader and
// records writes, which the unit runs at Commit.
type DummyReadWriter struct {
	DummyReader

	unit *Unit
}

// Add records the insert of dummy. Commit gives dummy its ID.
func (d *DummyReadWriter) Add(dummy *model.Dummy) {
	d.unit.add(func(ctx context.Context, queries *sqlc.Queries) error {
		id, err := queries.CreateDummy(ctx, dummy.Text)
		if err != nil {
			return fmt.Errorf("storing the dummy: %w", err)
		}

		dummy.ID = id

		return nil
	})
}
