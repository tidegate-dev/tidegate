// Package store reads and writes the authority's data in Postgres. It is the
// only package that runs SQL against the authority's tables.
package store

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
)

// Store gives access to the authority's tables.
type Store struct {
	pool *pgxpool.Pool
}

// New returns a Store on pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Users returns the queries for the users table.
func (s *Store) Users() *Users {
	return &Users{queries: sqlc.New(s.pool)}
}

// NewUnit returns an empty unit of work.
func (s *Store) NewUnit() *Unit {
	return &Unit{pool: s.pool}
}
