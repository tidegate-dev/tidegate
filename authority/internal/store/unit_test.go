//go:build integration

package store

import (
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/phemera-dev/phemera/authority/internal/model"
)

var _ = Describe("Unit", func() {
	var (
		pool *pgxpool.Pool
		data *Store
	)

	BeforeEach(func(ctx SpecContext) {
		pool, data = newStore(ctx)
	})

	It("writes nothing before Commit", func(ctx SpecContext) {
		unit := data.NewUnit()
		unit.Users().Add(newUser("alice"))

		Expect(count(ctx, pool, "users")).To(Equal(0))
	})

	It("runs the writes in the order they were added", func(ctx SpecContext) {
		alice := newUser("alice")

		unit := data.NewUnit()
		unit.Users().Add(alice)
		unit.Sessions().Add(newSession(alice.ID, "token"))

		Expect(unit.Commit(ctx)).To(Succeed())
		Expect(count(ctx, pool, "sessions")).To(Equal(1))
	})

	It("gives a stored row its version", func(ctx SpecContext) {
		alice := newUser("alice")

		unit := data.NewUnit()
		unit.Users().Add(alice)
		Expect(unit.Commit(ctx)).To(Succeed())

		found, err := data.Users().FindByName(ctx, "alice")
		Expect(err).NotTo(HaveOccurred())
		Expect(alice.Version).To(SatisfyAll(Not(BeZero()), Equal(found.Version)))
	})

	It("keeps no write if one fails", func(ctx SpecContext) {
		addUser(ctx, data, "alice")

		unit := data.NewUnit()
		unit.Users().Add(newUser("bob"))
		unit.Users().Add(newUser("alice"))

		Expect(unit.Commit(ctx)).To(MatchError(model.ErrUserNameTaken))
		Expect(count(ctx, pool, "users")).To(Equal(1))
	})

	It("refuses a second Commit", func(ctx SpecContext) {
		unit := data.NewUnit()
		unit.Users().Add(newUser("alice"))
		Expect(unit.Commit(ctx)).To(Succeed())

		Expect(unit.Commit(ctx)).To(MatchError(errUnitCommitted))
		Expect(count(ctx, pool, "users")).To(Equal(1))
	})
})
