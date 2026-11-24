//go:build integration

package postgres_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/dbtest"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "postgres secret engine integration")
}

// newPool returns a pool on an empty database.
func newPool(ctx context.Context) *pgxpool.Pool {
	GinkgoHelper()

	pool, err := pgxpool.New(ctx, dbtest.NewDatabase())
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(pool.Close)

	return pool
}
