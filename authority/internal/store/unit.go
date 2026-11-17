package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
	"github.com/tidegate-dev/tidegate/authority/internal/model"
)

var errUnitCommitted = errors.New("the unit of work was already committed")

// write is one pending change. It runs its queries inside the transaction of
// Commit.
type write func(ctx context.Context, queries *sqlc.Queries) error

// Unit is a unit of work. It collects writes and runs them in one transaction
// at Commit. Until then it writes nothing.
type Unit struct {
	pool      *pgxpool.Pool
	writes    []write
	committed bool
}

// Users returns the users table for this unit. Its reads run right away, its
// writes at Commit.
func (u *Unit) Users() model.UserReadWriter {
	return &UserReadWriter{queries: sqlc.New(u.pool), unit: u}
}

// Sessions returns the sessions table for this unit. Its reads run right
// away, its writes at Commit.
func (u *Unit) Sessions() model.SessionReadWriter {
	return &SessionReadWriter{queries: sqlc.New(u.pool), unit: u}
}

// Dummies returns the dummies table for this unit. Its reads run right away,
// its writes at Commit.
func (u *Unit) Dummies() model.DummyReadWriter {
	return &DummyReadWriter{queries: sqlc.New(u.pool), unit: u}
}

// Commit runs the pending writes in the order they were added, in one
// transaction. If one fails, none is kept. It returns
// model.ErrChangedSinceRead if a row changed after it was read. A unit can be
// committed once.
func (u *Unit) Commit(ctx context.Context) error {
	if u.committed {
		return errUnitCommitted
	}

	u.committed = true

	return pgx.BeginFunc(ctx, u.pool, func(tx pgx.Tx) error {
		queries := sqlc.New(tx)

		for _, run := range u.writes {
			err := run(ctx, queries)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (u *Unit) add(w write) {
	u.writes = append(u.writes, w)
}

// A version check that matched no row can't tell a changed row from a deleted one.
func changedIfExists(
	ctx context.Context,
	exists func(context.Context, pgtype.UUID) (bool, error),
	id pgtype.UUID,
	row string,
	gone error,
) error {
	found, err := exists(ctx, id)
	if err != nil {
		return fmt.Errorf("checking that the %s exists: %w", row, err)
	}

	if found {
		return model.ErrChangedSinceRead
	}

	return gone
}

func isUniqueViolation(err error) bool {
	return hasCode(err, pgerrcode.UniqueViolation)
}

func isForeignKeyViolation(err error) bool {
	return hasCode(err, pgerrcode.ForeignKeyViolation)
}

func hasCode(err error, code string) bool {
	var pgErr *pgconn.PgError

	return errors.As(err, &pgErr) && pgErr.Code == code
}
