//go:build integration

package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/dbtest"
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
