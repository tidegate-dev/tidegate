//go:build integration

// Package storetest gives Ginkgo specs a store on an empty Postgres database
// of their own.
package storetest

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/phemera-dev/phemera/authority/internal/db"
	"github.com/phemera-dev/phemera/authority/internal/dbtest"
	"github.com/phemera-dev/phemera/authority/internal/model"
	"github.com/phemera-dev/phemera/authority/internal/store"
)

// New returns a store on an empty, migrated database, and the pool the store
// uses. It closes the pool and drops the database when the spec ends.
func New(ctx context.Context) (*pgxpool.Pool, *store.Store) {
	GinkgoHelper()

	pool, err := db.Open(ctx, dbtest.NewDatabase())
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(pool.Close)
	Expect(db.Migrate(ctx, pool)).To(Succeed())

	return pool, store.New(pool)
}

// AddUser stores a user without a password and returns its ID.
func AddUser(ctx context.Context, units model.UnitFactory, name string) model.UserID {
	GinkgoHelper()

	id, err := model.NewUserID()
	Expect(err).NotTo(HaveOccurred())

	unit := units.NewUnit()
	unit.Users().Add(&model.User{ID: id, Name: name})
	Expect(unit.Commit(ctx)).To(Succeed())

	return id
}
