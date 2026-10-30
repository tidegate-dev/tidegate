//go:build integration

package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pressly/goose/v3/lock"

	"github.com/tidegate-dev/tidegate/authority/internal/config"
	"github.com/tidegate-dev/tidegate/authority/internal/db"
	"github.com/tidegate-dev/tidegate/authority/internal/dbtest"
)

var _ = Describe("openDatabase", func() {
	It("refuses an empty database", func(ctx SpecContext) {
		url := dbtest.NewDatabase()

		_, err := openDatabase(ctx, config.Database{URL: url})

		Expect(err).To(MatchError(db.ErrSchemaTooOld))
		Expect(err).To(MatchError(ContainSubstring("Run authority migrate first")))
	})

	It("opens a migrated database", func(ctx SpecContext) {
		url := dbtest.NewDatabase()
		Expect(migrate(ctx, config.Database{URL: url})).To(Succeed())

		pool, err := openDatabase(ctx, config.Database{URL: url})

		Expect(err).NotTo(HaveOccurred())
		pool.Close()
	})

	It("migrates once when two authorities wait for the lock", func(ctx SpecContext) {
		url := dbtest.NewDatabase()
		holder := connectToDatabase(ctx, url)
		takeMigrationLock(ctx, holder)

		first := startAuthority(ctx, url)
		second := startAuthority(ctx, url)
		Eventually(ctx, lockWaiters).WithArguments(holder).WithTimeout(30*time.Second).
			Should(Equal(2), "the authorities never waited for goose's migration lock")

		releaseMigrationLock(ctx, holder)

		Expect(<-first).To(Succeed())
		Expect(<-second).To(Succeed())
		Expect(maxTimesAMigrationWasApplied(ctx, holder)).To(Equal(1))
	}, SpecTimeout(time.Minute))
})

// startAuthority opens the database like an authority that migrates when it
// starts, and sends the result on the returned channel.
func startAuthority(ctx context.Context, url string) <-chan error {
	opened := make(chan error, 1)

	go func() {
		pool, err := openDatabase(ctx, config.Database{URL: url, MigrateOnStart: true})
		if err == nil {
			pool.Close()
		}

		opened <- err
	}()

	return opened
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

func connectToDatabase(ctx context.Context, url string) *pgx.Conn {
	GinkgoHelper()

	conn, err := pgx.Connect(ctx, url)
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() {
		_ = conn.Close(context.Background())
	})

	return conn
}

// lockWaiters counts the other sessions that have tried to take goose's
// lock. Between goose's attempts, pg_stat_activity shows the last attempt
// as the session's query.
func lockWaiters(ctx context.Context, conn *pgx.Conn) (int, error) {
	var waiters int

	err := conn.QueryRow(ctx,
		"SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND pid <> pg_backend_pid() AND query LIKE '%pg_try_advisory_lock%'",
	).Scan(&waiters)

	return waiters, err
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
