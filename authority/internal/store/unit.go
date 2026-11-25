package store

import (
	"context"
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

// Projects returns the projects table for this unit. Its writes run at
// Commit.
func (u *Unit) Projects() model.ProjectReadWriter {
	return &ProjectReadWriter{unit: u}
}

// Roles returns the roles table for this unit. Its writes run at Commit.
func (u *Unit) Roles() model.RoleReadWriter {
	return &RoleReadWriter{unit: u}
}

// Assets returns the assets table for this unit. Its writes run at Commit.
func (u *Unit) Assets() model.AssetReadWriter {
	return &AssetReadWriter{unit: u}
}

// Bindings returns the bindings table for this unit. Its writes run at
// Commit.
func (u *Unit) Bindings() model.BindingReadWriter {
	return &BindingReadWriter{unit: u}
}

// Tickets returns the tickets table for this unit. Its writes run at Commit.
func (u *Unit) Tickets() model.TicketReadWriter {
	return &TicketReadWriter{unit: u}
}

// Commit runs the pending writes in the order they were added, in one
// transaction. If one fails, none is kept. It returns
// model.ErrChangedSinceRead if a row changed after it was read. If it fails,
// the models it wrote may hold versions that were never stored, so read them
// again before you write them. A unit can be committed once.
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

func isUniqueViolation(err error) bool {
	return hasCode(err, pgerrcode.UniqueViolation)
}

func isUniqueViolationOf(err error, index string) bool {
	var pgErr *pgconn.PgError

	return isUniqueViolation(err) && errors.As(err, &pgErr) && pgErr.ConstraintName == index
}

func isForeignKeyViolation(err error) bool {
	return hasCode(err, pgerrcode.ForeignKeyViolation)
}

func isForeignKeyViolationOf(err error, constraint string) bool {
	var pgErr *pgconn.PgError

	return isForeignKeyViolation(err) && errors.As(err, &pgErr) && pgErr.ConstraintName == constraint
}

func hasCode(err error, code string) bool {
	var pgErr *pgconn.PgError

	return errors.As(err, &pgErr) && pgErr.Code == code
}
