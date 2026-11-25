//go:build integration

package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pressly/goose/v3/lock"

	"github.com/phemera-dev/phemera/authority/internal/dbtest"
)

var _ = Describe("Migrate", func() {
	It("changes nothing the second time", func(ctx SpecContext) {
		pool, err := Open(ctx, dbtest.NewDatabase())
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(pool.Close)

		Expect(Migrate(ctx, pool)).To(Succeed())
		before := appliedMigrations(ctx, pool)

		Expect(Migrate(ctx, pool)).To(Succeed())

		Expect(appliedMigrations(ctx, pool)).To(Equal(before))
	})

	It("leaves no dummies table", func(ctx SpecContext) {
		pool, err := Open(ctx, dbtest.NewDatabase())
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(pool.Close)

		Expect(Migrate(ctx, pool)).To(Succeed())

		var table *string
		Expect(pool.QueryRow(ctx, "SELECT to_regclass('dummies')::text").Scan(&table)).To(Succeed())
		Expect(table).To(BeNil())
	})

	It("migrates once when two authorities wait for the lock", func(ctx SpecContext) {
		url := dbtest.NewDatabase()
		holder := dbtest.Connect(ctx, url)
		takeMigrationLock(ctx, holder)

		firstMigrationErrorChan := startMigration(ctx, url)
		secondMigrationErrorChan := startMigration(ctx, url)
		Eventually(ctx, countSessionsTryingMigrationLock).WithArguments(holder).WithTimeout(30*time.Second).
			Should(Equal(2), "the authorities never waited for goose's migration lock")

		releaseMigrationLock(ctx, holder)

		Expect(<-firstMigrationErrorChan).To(Succeed())
		Expect(<-secondMigrationErrorChan).To(Succeed())
		Expect(maxTimesAMigrationWasApplied(ctx, holder)).To(Equal(1))
	}, SpecTimeout(time.Minute))
})

var _ = Describe("OpenMigrated", func() {
	It("refuses an empty database", func(ctx SpecContext) {
		url := dbtest.NewDatabase()

		_, err := OpenMigrated(ctx, url)

		Expect(err).To(MatchError(ErrSchemaTooOld))
	})

	It("opens a migrated database", func(ctx SpecContext) {
		url := dbtest.NewDatabase()
		Expect(migrateAt(ctx, url)).To(Succeed())

		pool, err := OpenMigrated(ctx, url)

		Expect(err).NotTo(HaveOccurred())
		pool.Close()
	})
})

type appliedMigration struct {
	Version   int64
	AppliedAt time.Time
}

func appliedMigrations(ctx context.Context, pool *pgxpool.Pool) []appliedMigration {
	GinkgoHelper()

	rows, err := pool.Query(ctx, "SELECT version_id, tstamp FROM goose_db_version ORDER BY id")
	Expect(err).NotTo(HaveOccurred())

	migrations, err := pgx.CollectRows(rows, pgx.RowToStructByPos[appliedMigration])
	Expect(err).NotTo(HaveOccurred())

	return migrations
}

// startMigration migrates the database in the background and sends the
// migration's error on the returned channel.
func startMigration(ctx context.Context, url string) <-chan error {
	migrationErrorChan := make(chan error, 1)

	go func() {
		migrationErrorChan <- migrateAt(ctx, url)
	}()

	return migrationErrorChan
}

// Each authority migrates on a connection pool of its own.
func migrateAt(ctx context.Context, url string) error {
	pool, err := Open(ctx, url)
	if err != nil {
		return err
	}
	defer pool.Close()

	return Migrate(ctx, pool)
}

func takeMigrationLock(ctx context.Context, conn *pgx.Conn) {
	GinkgoHelper()

	_, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", lock.DefaultLockID)
	Expect(err).NotTo(HaveOccurred())
}

func releaseMigrationLock(ctx context.Context, conn *pgx.Conn) {
	GinkgoHelper()

	_, err := conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", lock.DefaultLockID)
	Expect(err).NotTo(HaveOccurred())
}

func countSessionsTryingMigrationLock(ctx context.Context, conn *pgx.Conn) (int, error) {
	var sessions int

	// Between goose's attempts, pg_stat_activity shows the last attempt as
	// the session's query.
	err := conn.QueryRow(ctx,
		"SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND pid <> pg_backend_pid() AND query LIKE '%pg_try_advisory_lock%'",
	).Scan(&sessions)

	return sessions, err
}

// maxTimesAMigrationWasApplied counts goose's rows for each migration and
// returns the highest count.
func maxTimesAMigrationWasApplied(ctx context.Context, conn *pgx.Conn) int {
	GinkgoHelper()

	var count int

	err := conn.QueryRow(ctx,
		"SELECT max(applied) FROM (SELECT count(*) AS applied FROM goose_db_version WHERE version_id > 0 GROUP BY version_id) AS migrations",
	).Scan(&count)
	Expect(err).NotTo(HaveOccurred())

	return count
}
