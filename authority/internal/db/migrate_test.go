//go:build integration

package db

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/authority/internal/dbtest"
)

func TestMigrateTwiceChangesNothingTheSecondTime(t *testing.T) {
	t.Parallel()

	// arrange
	pool, err := Open(t.Context(), dbtest.NewDatabase(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	require.NoError(t, Migrate(t.Context(), pool))
	before := appliedMigrations(t, pool)

	// act
	err = Migrate(t.Context(), pool)

	// assert
	require.NoError(t, err)
	assert.Equal(t, before, appliedMigrations(t, pool))
}

type appliedMigration struct {
	Version   int64
	AppliedAt time.Time
}

func appliedMigrations(t *testing.T, pool *pgxpool.Pool) []appliedMigration {
	t.Helper()

	rows, err := pool.Query(t.Context(), "SELECT version_id, tstamp FROM goose_db_version ORDER BY id")
	require.NoError(t, err)

	migrations, err := pgx.CollectRows(rows, pgx.RowToStructByPos[appliedMigration])
	require.NoError(t, err)

	return migrations
}
