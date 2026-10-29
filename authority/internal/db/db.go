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
	"path"

	"github.com/jackc/pgx/v5/pgconn"
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

// The SQLSTATE code for a table that doesn't exist.
const undefinedTable = "42P01"

// Open connects to the database at url. It warns if the connection may run
// unencrypted.
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		// pgx masks the password only where it can find it, and a malformed URL
		// can hide it. pgx also reads the PG variables.
		return nil, errors.New("the database URL isn't a valid Postgres connection URL, or a PG variable like PGSSLMODE is invalid")
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
func Migrate(ctx context.Context, pool *pgxpool.Pool) (err error) {
	provider, err := newProvider(pool)
	if err != nil {
		return err
	}

	defer func() {
		err = errors.Join(err, provider.Close())
	}()

	slog.Info("migrating the database, after any other authority that's migrating it")

	results, err := provider.Up(ctx)
	for _, result := range results {
		slog.Info("applied a migration",
			"version", result.Source.Version,
			"file", path.Base(result.Source.Path),
			"duration", result.Duration)
	}

	if err != nil {
		return fmt.Errorf("migrating the database: %w", err)
	}

	if len(results) == 0 {
		slog.Info("the database schema is up to date")
	}

	return nil
}

// CheckSchema returns ErrSchemaTooOld if the database misses migrations. A
// newer schema is fine, because migrations only add to it. It only reads from
// the database.
func CheckSchema(ctx context.Context, pool *pgxpool.Pool) (err error) {
	current, err := schemaVersion(ctx, pool)
	if err != nil {
		return err
	}

	provider, err := newProvider(pool)
	if err != nil {
		return err
	}

	defer func() {
		err = errors.Join(err, provider.Close())
	}()

	sources := provider.ListSources()
	target := sources[len(sources)-1].Version

	if current < target {
		return fmt.Errorf("%w: it's at version %d, the authority needs %d", ErrSchemaTooOld, current, target)
	}

	return nil
}

// schemaVersion reads the version from goose's table itself, because goose
// creates the table before reading it, and that needs more than read access.
func schemaVersion(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var version *int64

	err := pool.QueryRow(ctx, "SELECT max(version_id) FROM "+goose.DefaultTablename).Scan(&version)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == undefinedTable {
		return 0, nil
	}

	if err != nil {
		return 0, fmt.Errorf("reading the schema version: %w", err)
	}

	if version == nil {
		return 0, nil
	}

	return *version, nil
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
