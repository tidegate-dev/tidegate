//go:build integration

package store

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
)

const sessionLifetime = time.Hour

var issuedAt = time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)

var _ = Describe("SessionReader", func() {
	var (
		data  *Store
		alice *model.User
	)

	BeforeEach(func(ctx SpecContext) {
		_, data = newStore(ctx)
		alice = addUser(ctx, data, "alice")
	})

	Describe("ActiveByTokenHash", func() {
		It("returns the session of the token hash", func(ctx SpecContext) {
			session := addSession(ctx, data, alice.ID, "token")

			found, err := data.Sessions().ActiveByTokenHash(ctx, tokenHash("token"), issuedAt)

			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(SatisfyAll(
				HaveField("ID", session.ID),
				HaveField("UserID", alice.ID),
				HaveField("CreatedAt", BeTemporally("==", issuedAt)),
				HaveField("ExpiresAt", BeTemporally("==", issuedAt.Add(sessionLifetime))),
				HaveField("Version", session.Version),
			))
			Expect(found.TokenHash.Reveal()).To(Equal(tokenHash("token").Reveal()))
		})

		It("returns ErrSessionNotFound for a token hash that no session has", func(ctx SpecContext) {
			addSession(ctx, data, alice.ID, "token")

			_, err := data.Sessions().ActiveByTokenHash(ctx, tokenHash("other"), issuedAt)

			Expect(err).To(MatchError(model.ErrSessionNotFound))
		})

		It("returns the session until just before it expires", func(ctx SpecContext) {
			addSession(ctx, data, alice.ID, "token")

			_, err := data.Sessions().ActiveByTokenHash(ctx, tokenHash("token"), issuedAt.Add(sessionLifetime-time.Microsecond))

			Expect(err).NotTo(HaveOccurred())
		})

		It("returns ErrSessionNotFound from the time the session expires", func(ctx SpecContext) {
			addSession(ctx, data, alice.ID, "token")

			_, err := data.Sessions().ActiveByTokenHash(ctx, tokenHash("token"), issuedAt.Add(sessionLifetime))

			Expect(err).To(MatchError(model.ErrSessionNotFound))
		})
	})
})

var _ = Describe("SessionReadWriter", func() {
	var (
		pool  *pgxpool.Pool
		data  *Store
		alice *model.User
	)

	BeforeEach(func(ctx SpecContext) {
		pool, data = newStore(ctx)
		alice = addUser(ctx, data, "alice")
	})

	Describe("Add", func() {
		It("refuses a token hash that another session has", func(ctx SpecContext) {
			addSession(ctx, data, alice.ID, "token")

			unit := data.NewUnit()
			unit.Sessions().Add(newSession(alice.ID, "token"))

			Expect(unit.Commit(ctx)).To(MatchError(ContainSubstring("token hash already in use")))
			Expect(count(ctx, pool, "sessions")).To(Equal(1))
		})

		It("returns ErrUserNotFound for a user that is gone", func(ctx SpecContext) {
			unit := data.NewUnit()
			unit.Sessions().Add(newSession(newUser("bob").ID, "token"))

			Expect(unit.Commit(ctx)).To(MatchError(model.ErrUserNotFound))
		})
	})

	Describe("Remove", func() {
		It("deletes only the session", func(ctx SpecContext) {
			session := addSession(ctx, data, alice.ID, "token")
			addSession(ctx, data, alice.ID, "other")

			unit := data.NewUnit()
			unit.Sessions().Remove(session)

			Expect(unit.Commit(ctx)).To(Succeed())
			Expect(count(ctx, pool, "sessions")).To(Equal(1))

			_, err := data.Sessions().ActiveByTokenHash(ctx, tokenHash("other"), issuedAt)
			Expect(err).NotTo(HaveOccurred())
		})

		It("deletes a session that changed since it was read", func(ctx SpecContext) {
			session := addSession(ctx, data, alice.ID, "token")
			_, err := pool.Exec(ctx, "UPDATE sessions SET expires_at = expires_at + interval '1 hour'")
			Expect(err).NotTo(HaveOccurred())

			unit := data.NewUnit()
			unit.Sessions().Remove(session)

			Expect(unit.Commit(ctx)).To(Succeed())
			Expect(count(ctx, pool, "sessions")).To(Equal(0))
		})

		It("succeeds for a session that is already gone", func(ctx SpecContext) {
			unit := data.NewUnit()
			unit.Sessions().Remove(newSession(alice.ID, "token"))

			Expect(unit.Commit(ctx)).To(Succeed())
		})
	})

	Describe("RemoveOfUser", func() {
		It("deletes every session of the user, and no other user's", func(ctx SpecContext) {
			bob := addUser(ctx, data, "bob")
			addSession(ctx, data, alice.ID, "first")
			addSession(ctx, data, alice.ID, "second")
			addSession(ctx, data, bob.ID, "bobs")

			unit := data.NewUnit()
			unit.Sessions().RemoveOfUser(alice.ID)

			Expect(unit.Commit(ctx)).To(Succeed())
			Expect(count(ctx, pool, "sessions")).To(Equal(1))

			_, err := data.Sessions().ActiveByTokenHash(ctx, tokenHash("bobs"), issuedAt)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("RemoveExpiredOfUser", func() {
		It("deletes the user's sessions that expire at the given time or earlier", func(ctx SpecContext) {
			bob := addUser(ctx, data, "bob")
			addSession(ctx, data, alice.ID, "expired")
			addSession(ctx, data, bob.ID, "bobs")

			later := newSession(alice.ID, "active")
			later.ExpiresAt = later.ExpiresAt.Add(time.Microsecond)

			unit := data.NewUnit()
			unit.Sessions().Add(later)
			unit.Sessions().RemoveExpiredOfUser(alice.ID, issuedAt.Add(sessionLifetime))

			Expect(unit.Commit(ctx)).To(Succeed())
			Expect(count(ctx, pool, "sessions")).To(Equal(2))

			_, err := data.Sessions().ActiveByTokenHash(ctx, tokenHash("active"), issuedAt)
			Expect(err).NotTo(HaveOccurred())

			_, err = data.Sessions().ActiveByTokenHash(ctx, tokenHash("bobs"), issuedAt)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
