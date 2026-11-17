//go:build integration

package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

var _ = Describe("UserReader", func() {
	var data *Store

	BeforeEach(func(ctx SpecContext) {
		_, data = newStore(ctx)
	})

	Describe("ByName", func() {
		It("returns the user, ignoring the letter case of the name", func(ctx SpecContext) {
			alice := addUser(ctx, data, "alice")

			found, err := data.Users().ByName(ctx, "ALICE")

			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(Equal(alice))
		})

		It("returns the password hash of a user that has one", func(ctx SpecContext) {
			hash := secret.New("a hash")
			alice := newUser("alice")
			alice.PasswordHash = &hash

			unit := data.NewUnit()
			unit.Users().Add(alice)
			Expect(unit.Commit(ctx)).To(Succeed())

			found, err := data.Users().ByName(ctx, "alice")

			Expect(err).NotTo(HaveOccurred())
			Expect(found.PasswordHash.Reveal()).To(Equal("a hash"))
		})

		It("returns ErrUserNotFound for a name that no user has", func(ctx SpecContext) {
			addUser(ctx, data, "alice")

			_, err := data.Users().ByName(ctx, "bob")

			Expect(err).To(MatchError(model.ErrUserNotFound))
		})
	})

	Describe("Any", func() {
		It("reports false for an empty table", func(ctx SpecContext) {
			Expect(data.Users().Any(ctx)).To(BeFalse())
		})

		It("reports true once a user exists", func(ctx SpecContext) {
			addUser(ctx, data, "alice")

			Expect(data.Users().Any(ctx)).To(BeTrue())
		})
	})
})

var _ = Describe("UserReadWriter", func() {
	var (
		pool *pgxpool.Pool
		data *Store
	)

	BeforeEach(func(ctx SpecContext) {
		pool, data = newStore(ctx)
	})

	Describe("Add", func() {
		It("refuses a name that another user has in another letter case", func(ctx SpecContext) {
			addUser(ctx, data, "alice")

			unit := data.NewUnit()
			unit.Users().Add(newUser("Alice"))

			Expect(unit.Commit(ctx)).To(MatchError(model.ErrConflict))
			Expect(count(ctx, pool, "users")).To(Equal(1))
		})
	})

	Describe("AddIfNoneExist", func() {
		It("stores the first user", func(ctx SpecContext) {
			unit := data.NewUnit()
			unit.Users().AddIfNoneExist(newUser("alice"))

			Expect(unit.Commit(ctx)).To(Succeed())
			Expect(count(ctx, pool, "users")).To(Equal(1))
		})

		It("refuses when a user exists", func(ctx SpecContext) {
			addUser(ctx, data, "alice")

			unit := data.NewUnit()
			unit.Users().AddIfNoneExist(newUser("bob"))

			Expect(unit.Commit(ctx)).To(MatchError(model.ErrUsersExist))
			Expect(count(ctx, pool, "users")).To(Equal(1))
		})

		It("stores one user when two units wait for the lock", func(ctx SpecContext) {
			holder, err := pool.Acquire(ctx)
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(holder.Release)
			_, err = holder.Exec(ctx, "SELECT pg_advisory_lock($1)", usersEmptyLockID)
			Expect(err).NotTo(HaveOccurred())

			first := startAddIfNoneExist(ctx, data, "alice")
			second := startAddIfNoneExist(ctx, data, "bob")
			Eventually(ctx, countWaitingForUsersEmptyLock).WithArguments(pool).WithTimeout(30*time.Second).
				Should(Equal(2), "the units never waited for the lock")

			_, err = holder.Exec(ctx, "SELECT pg_advisory_unlock($1)", usersEmptyLockID)
			Expect(err).NotTo(HaveOccurred())

			Expect([]error{<-first, <-second}).To(ConsistOf(Succeed(), MatchError(model.ErrUsersExist)))
			Expect(count(ctx, pool, "users")).To(Equal(1))
		}, SpecTimeout(time.Minute))
	})

	Describe("Update", func() {
		It("stores the changed fields and the new version", func(ctx SpecContext) {
			alice := addUser(ctx, data, "alice")
			hash := secret.New("a hash")
			alice.Name, alice.PasswordHash = "alicia", &hash

			unit := data.NewUnit()
			unit.Users().Update(alice)
			Expect(unit.Commit(ctx)).To(Succeed())

			found, err := data.Users().ByName(ctx, "alicia")
			Expect(err).NotTo(HaveOccurred())
			Expect(found.PasswordHash.Reveal()).To(Equal("a hash"))
			Expect(found.Version).To(Equal(alice.Version))
		})

		It("refuses a user that changed since it was read", func(ctx SpecContext) {
			addUser(ctx, data, "alice")

			stale, err := data.Users().ByName(ctx, "alice")
			Expect(err).NotTo(HaveOccurred())

			fresh, err := data.Users().ByName(ctx, "alice")
			Expect(err).NotTo(HaveOccurred())

			fresh.Name = "alicia"
			first := data.NewUnit()
			first.Users().Update(fresh)
			Expect(first.Commit(ctx)).To(Succeed())

			stale.Name = "ali"
			second := data.NewUnit()
			second.Users().Update(stale)

			Expect(second.Commit(ctx)).To(MatchError(model.ErrConflict))
			Expect(data.Users().ByName(ctx, "alicia")).NotTo(BeNil())
		})

		It("refuses a name that another user has", func(ctx SpecContext) {
			addUser(ctx, data, "alice")
			bob := addUser(ctx, data, "bob")
			bob.Name = "alice"

			unit := data.NewUnit()
			unit.Users().Update(bob)

			Expect(unit.Commit(ctx)).To(MatchError(model.ErrConflict))
		})

		It("returns ErrUserNotFound for a user that is gone", func(ctx SpecContext) {
			unit := data.NewUnit()
			unit.Users().Update(newUser("alice"))

			Expect(unit.Commit(ctx)).To(MatchError(model.ErrUserNotFound))
		})
	})

	Describe("Remove", func() {
		It("deletes the user", func(ctx SpecContext) {
			alice := addUser(ctx, data, "alice")

			unit := data.NewUnit()
			unit.Users().Remove(alice)

			Expect(unit.Commit(ctx)).To(Succeed())
			Expect(count(ctx, pool, "users")).To(Equal(0))
		})

		It("refuses a user that changed since it was read", func(ctx SpecContext) {
			alice := addUser(ctx, data, "alice")
			stale := *alice

			alice.Name = "alicia"
			first := data.NewUnit()
			first.Users().Update(alice)
			Expect(first.Commit(ctx)).To(Succeed())

			second := data.NewUnit()
			second.Users().Remove(&stale)

			Expect(second.Commit(ctx)).To(MatchError(model.ErrConflict))
			Expect(count(ctx, pool, "users")).To(Equal(1))
		})

		It("succeeds for a user that is already gone", func(ctx SpecContext) {
			unit := data.NewUnit()
			unit.Users().Remove(newUser("alice"))

			Expect(unit.Commit(ctx)).To(Succeed())
		})
	})
})

func startAddIfNoneExist(ctx context.Context, data *Store, name string) <-chan error {
	GinkgoHelper()

	user := newUser(name)
	committed := make(chan error, 1)

	go func() {
		unit := data.NewUnit()
		unit.Users().AddIfNoneExist(user)

		committed <- unit.Commit(ctx)
	}()

	return committed
}

func countWaitingForUsersEmptyLock(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var waiting int

	// pg_locks splits a bigint advisory key into objid (low 32 bits) and
	// classid (high 32 bits).
	err := pool.QueryRow(ctx,
		"SELECT count(*) FROM pg_locks WHERE locktype = 'advisory' AND NOT granted AND ((classid::bigint << 32) | objid::bigint) = $1",
		usersEmptyLockID,
	).Scan(&waiting)

	return waiting, err
}
