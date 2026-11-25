// Package postgres is a secret engine that stores its values in a Postgres
// schema of its own. It stores them unencrypted.
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Postgres can fail one of two CREATE SCHEMA IF NOT EXISTS that run at once.
const lockSchemaCreation = `SELECT pg_advisory_xact_lock(7214365028173645912)`

const createSchema = `CREATE SCHEMA IF NOT EXISTS secret_engine`

const createSecretsTable = `
CREATE TABLE IF NOT EXISTS secret_engine.secrets (
    id uuid CONSTRAINT secrets_pk PRIMARY KEY,
    value bytea NOT NULL
)`

// CreateSchema creates the engine's Postgres schema and its table if the
// database doesn't have them yet.
func CreateSchema(ctx context.Context, pool *pgxpool.Pool) error {
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		for _, statement := range []string{lockSchemaCreation, createSchema, createSecretsTable} {
			_, err := tx.Exec(ctx, statement)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("creating the schema of the Postgres secret engine: %w", err)
	}

	return nil
}
