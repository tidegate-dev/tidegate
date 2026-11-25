//go:build integration

package postgres_test

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/secretengine/postgres"
)

var _ = Describe("CreateSchema", func() {
	It("creates the secrets table in an empty database", func(ctx SpecContext) {
		pool := newPool(ctx)

		Expect(postgres.CreateSchema(ctx, pool)).To(Succeed())

		Expect(secretsTableExists(ctx, pool)).To(BeTrue())
	})

	It("gives the ID column of the secrets table no default", func(ctx SpecContext) {
		pool := newPool(ctx)

		Expect(postgres.CreateSchema(ctx, pool)).To(Succeed())

		var columnDefault *string

		Expect(pool.QueryRow(ctx,
			`SELECT column_default FROM information_schema.columns
			 WHERE table_schema = 'secret_engine' AND table_name = 'secrets' AND column_name = 'id'`,
		).Scan(&columnDefault)).To(Succeed())
		Expect(columnDefault).To(BeNil())
	})

	It("succeeds in a database that has the secrets table", func(ctx SpecContext) {
		pool := newPool(ctx)
		Expect(postgres.CreateSchema(ctx, pool)).To(Succeed())

		Expect(postgres.CreateSchema(ctx, pool)).To(Succeed())
	})

	It("succeeds when several callers create the schema at the same time", func(ctx SpecContext) {
		const callers = 8

		pool := newPool(ctx)
		errChan := make(chan error, callers)

		for range callers {
			go func() {
				errChan <- postgres.CreateSchema(ctx, pool)
			}()
		}

		for range callers {
			Expect(<-errChan).To(Succeed())
		}
	})
})

func secretsTableExists(ctx context.Context, pool *pgxpool.Pool) bool {
	GinkgoHelper()

	var exists bool

	err := pool.QueryRow(ctx, "SELECT to_regclass('secret_engine.secrets') IS NOT NULL").Scan(&exists)
	Expect(err).NotTo(HaveOccurred())

	return exists
}
