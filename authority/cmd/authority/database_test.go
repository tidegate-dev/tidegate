//go:build integration

package main

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pressly/goose/v3/lock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/authority/internal/config"
	"github.com/tidegate-dev/tidegate/authority/internal/db"
	"github.com/tidegate-dev/tidegate/authority/internal/dbtest"
)

func TestOpenDatabaseRefusesAnEmptyDatabase(t *testing.T) {
	t.Parallel()

	// arrange
	url := dbtest.NewDatabase(t)

	// act
	_, err := openDatabase(t.Context(), config.Database{URL: url})

	// assert
	require.ErrorIs(t, err, db.ErrSchemaTooOld)
	assert.ErrorContains(t, err, "Run authority migrate first")
}

func TestOpenDatabaseOpensAMigratedDatabase(t *testing.T) {
	t.Parallel()

	// arrange
	url := dbtest.NewDatabase(t)
	require.NoError(t, migrate(t.Context(), config.Database{URL: url}))

	// act
	pool, err := openDatabase(t.Context(), config.Database{URL: url})

	// assert
	require.NoError(t, err)
	pool.Close()
}

func TestOpenDatabaseMigratesOnceWhenTwoAuthoritiesWaitForTheLock(t *testing.T) {
	t.Parallel()

	// arrange
	url := dbtest.NewDatabase(t)
	holder := connectToDatabase(t, url)
	_, err := holder.Exec(t.Context(), "SELECT pg_advisory_lock($1)", lock.DefaultLockID)
	require.NoError(t, err)

	cfg := config.Database{URL: url, MigrateOnStart: true}
	opened := make(chan error, 2)

	for range 2 {
		go func() {
			pool, err := openDatabase(t.Context(), cfg)
			if err == nil {
				pool.Close()
			}

			opened <- err
		}()
	}

	waitForLockWaiters(t, holder, 2)

	// act
	_, err = holder.Exec(t.Context(), "SELECT pg_advisory_unlock($1)", lock.DefaultLockID)
	require.NoError(t, err)

	// assert
	require.NoError(t, <-opened)
	require.NoError(t, <-opened)
	assert.Equal(t, 1, maxTimesAMigrationWasApplied(t, holder))
}

func connectToDatabase(t *testing.T, url string) *pgx.Conn {
	t.Helper()

	conn, err := pgx.Connect(t.Context(), url)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = conn.Close(context.Background())
	})

	return conn
}

// waitForLockWaiters waits until count other sessions have tried to take
// goose's lock. Between goose's attempts, pg_stat_activity shows the last
// attempt as the session's query.
func waitForLockWaiters(t *testing.T, conn *pgx.Conn, count int) {
	t.Helper()

	assert.Eventually(t, func() bool {
		var waiters int

		err := conn.QueryRow(t.Context(),
			"SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND pid <> pg_backend_pid() AND query LIKE '%pg_try_advisory_lock%'",
		).Scan(&waiters)

		// Eventually calls this function on another goroutine, where
		// require can't stop the test.
		return assert.NoError(t, err) && waiters == count
	}, 30*time.Second, 50*time.Millisecond, "%d sessions never waited for goose's migration lock", count)
}

// maxTimesAMigrationWasApplied counts goose's rows for each migration and
// returns the highest count.
func maxTimesAMigrationWasApplied(t *testing.T, conn *pgx.Conn) int {
	t.Helper()

	var count int

	err := conn.QueryRow(t.Context(),
		"SELECT max(applied) FROM (SELECT count(*) AS applied FROM goose_db_version WHERE version_id > 0 GROUP BY version_id) AS migrations",
	).Scan(&count)
	require.NoError(t, err)

	return count
}
