// Package store reads and writes the authority's data in Postgres. It is the
// only package that queries the authority's tables.
package store

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
	"github.com/tidegate-dev/tidegate/authority/internal/model"
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
func (s *Store) Users() *UserReader {
	return &UserReader{queries: sqlc.New(s.pool)}
}

// Sessions returns the queries for the sessions table.
func (s *Store) Sessions() *SessionReader {
	return &SessionReader{queries: sqlc.New(s.pool)}
}

// Dummies returns the queries for the dummies table.
func (s *Store) Dummies() *DummyReader {
	return &DummyReader{queries: sqlc.New(s.pool)}
}

// NewUnit returns an empty unit of work.
func (s *Store) NewUnit() model.Unit {
	return &Unit{pool: s.pool}
}
