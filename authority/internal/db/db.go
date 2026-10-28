// Package db connects the authority to Postgres and keeps its schema up to
// date.
package db

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

//go:embed migrations/*.sql
var migrations embed.FS

// ErrSchemaTooOld means the database misses migrations that this version of
// the authority needs.
var ErrSchemaTooOld = errors.New("the database schema is older than the authority expects")

// Open connects to the database at url. It warns if the connection may run
// unencrypted.
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parsing the database URL: %w", err)
	}

	if fallsBackToPlaintext(config) {
		slog.Warn("the connection to the database may run unencrypted, because its URL sets no sslmode. " +
			"Set sslmode=verify-full, or sslmode=disable if you don't want TLS")
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("connecting to the database: %w", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		pool.Close()

		return nil, fmt.Errorf("connecting to the database: %w", err)
	}

	return pool, nil
}

// Migrate applies every migration that the database doesn't have yet.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	provider, err := newProvider(pool)
	if err != nil {
		return err
	}

	_, err = provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("migrating the database: %w", err)
	}

	return nil
}

// CheckSchema returns ErrSchemaTooOld if the database misses migrations. A
// newer schema is fine, because migrations only add to it.
func CheckSchema(ctx context.Context, pool *pgxpool.Pool) error {
	provider, err := newProvider(pool)
	if err != nil {
		return err
	}

	current, target, err := provider.GetVersions(ctx)
	if err != nil {
		return fmt.Errorf("reading the schema version: %w", err)
	}

	if current < target {
		return fmt.Errorf("%w: it's at version %d, the authority needs %d", ErrSchemaTooOld, current, target)
	}

	return nil
}

// fallsBackToPlaintext reports whether the connection tries TLS and then
// plaintext. pgx doesn't keep the sslmode, and this is what its default,
// prefer, does.
func fallsBackToPlaintext(config *pgxpool.Config) bool {
	if config.ConnConfig.TLSConfig == nil {
		return false
	}

	for _, fallback := range config.ConnConfig.Fallbacks {
		if fallback.TLSConfig == nil {
			return true
		}
	}

	return false
}

func newProvider(pool *pgxpool.Pool) (*goose.Provider, error) {
	fsys, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("reading the migrations: %w", err)
	}

	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return nil, fmt.Errorf("creating the migration lock: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, stdlib.OpenDBFromPool(pool), fsys,
		goose.WithSessionLocker(locker))
	if err != nil {
		return nil, fmt.Errorf("loading the migrations: %w", err)
	}

	return provider, nil
}
