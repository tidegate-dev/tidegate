package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tidegate-dev/tidegate/internal/secret"
	"github.com/tidegate-dev/tidegate/internal/secretengine"
)

const insertValue = `INSERT INTO secret_engine.secrets (value) VALUES ($1) RETURNING id::text`

const selectValue = `SELECT value FROM secret_engine.secrets WHERE id = $1`

const deleteValue = `DELETE FROM secret_engine.secrets WHERE id = $1`

// Engine stores secret values in the table that CreateSchema creates. A
// reference is the ID of a value's row.
type Engine struct {
	pool *pgxpool.Pool
}

// New returns an Engine on pool.
func New(pool *pgxpool.Pool) *Engine {
	return &Engine{pool: pool}
}

// Store saves the value in a new row and returns the row's ID as the
// reference.
func (e *Engine) Store(ctx context.Context, value secret.Bytes) (secretengine.Reference, error) {
	var id string

	err := e.pool.QueryRow(ctx, insertValue, value.Reveal()).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("storing a value in the Postgres secret engine: %w", err)
	}

	return secretengine.Reference(id), nil
}

// Fetch returns the value in the row that the reference names. If there is no
// such row, it returns secretengine.ErrNotFound.
func (e *Engine) Fetch(ctx context.Context, reference secretengine.Reference) (secret.Bytes, error) {
	id, err := rowID(reference)
	if err != nil {
		return secret.Bytes{}, err
	}

	var value []byte

	err = e.pool.QueryRow(ctx, selectValue, id).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return secret.Bytes{}, secretengine.ErrNotFound
	}

	if err != nil {
		return secret.Bytes{}, fmt.Errorf("fetching a value from the Postgres secret engine: %w", err)
	}

	return secret.NewBytes(value), nil
}

// Delete removes the row that the reference names. If there is no such row,
// it returns secretengine.ErrNotFound.
func (e *Engine) Delete(ctx context.Context, reference secretengine.Reference) error {
	id, err := rowID(reference)
	if err != nil {
		return err
	}

	deleted, err := e.pool.Exec(ctx, deleteValue, id)
	if err != nil {
		return fmt.Errorf("deleting a value from the Postgres secret engine: %w", err)
	}

	if deleted.RowsAffected() == 0 {
		return secretengine.ErrNotFound
	}

	return nil
}

func rowID(reference secretengine.Reference) (pgtype.UUID, error) {
	var id pgtype.UUID

	err := id.Scan(string(reference))
	if err != nil {
		return pgtype.UUID{}, secretengine.ErrNotFound
	}

	return id, nil
}
