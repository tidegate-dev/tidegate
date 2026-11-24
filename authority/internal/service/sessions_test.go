//go:build integration

package service

import (
	"context"
	"crypto/sha256"
	"time"

	"github.com/The127/go-clock"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	"github.com/tidegate-dev/tidegate/authority/internal/store"
	"github.com/tidegate-dev/tidegate/authority/internal/storetest"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

type storedToken struct {
	TokenHash []byte
	UserID    [16]byte
	Lifetime  time.Duration
}

var _ = Describe("Issue", func() {
	var (
		pool   *pgxpool.Pool
		data   *store.Store
		userID model.UserID
	)

	BeforeEach(func(ctx SpecContext) {
		pool, data = storetest.New(ctx)
		userID = storetest.AddUser(ctx, data, "alice")
	})

	It("returns a token and stores only its hash", func(ctx SpecContext) {
		token, err := newSessions(data, 12*time.Hour).Issue(ctx, userID)

		Expect(err).NotTo(HaveOccurred())
		Expect(token.Reveal()).To(MatchRegexp(`^[A-Za-z0-9_-]{43}$`))

		hash := sha256.Sum256([]byte(token.Reveal()))
		Expect(storedTokens(ctx, pool)).To(ConsistOf(SatisfyAll(
			HaveField("TokenHash", Equal(hash[:])),
			HaveField("UserID", [16]byte(userID)),
			HaveField("Lifetime", 12*time.Hour),
		)))
	})

	It("returns a new token each time", func(ctx SpecContext) {
		sessions := newSessions(data, 12*time.Hour)

		first, err := sessions.Issue(ctx, userID)
		Expect(err).NotTo(HaveOccurred())

		second, err := sessions.Issue(ctx, userID)
		Expect(err).NotTo(HaveOccurred())

		Expect(second).NotTo(Equal(first))
		Expect(storedTokens(ctx, pool)).To(HaveLen(2))
	})

	It("stores the expiry one lifetime after issuing the token", func(ctx SpecContext) {
		_, err := newSessions(data, time.Second).Issue(ctx, userID)

		Expect(err).NotTo(HaveOccurred())
		Expect(storedTokens(ctx, pool)).To(ConsistOf(HaveField("Lifetime", time.Second)))
	})

	It("deletes the user's expired sessions", func(ctx SpecContext) {
		_, err := pool.Exec(ctx, `INSERT INTO sessions (token_hash, user_id, created_at, expires_at)
			VALUES (sha256('expired'), $1, now(), now() - interval '1 second')`, userID.String())
		Expect(err).NotTo(HaveOccurred())

		token, err := newSessions(data, 12*time.Hour).Issue(ctx, userID)

		Expect(err).NotTo(HaveOccurred())

		hash := sha256.Sum256([]byte(token.Reveal()))
		Expect(storedTokens(ctx, pool)).To(ConsistOf(HaveField("TokenHash", Equal(hash[:]))))
	})

	It("fails for a user that is gone", func(ctx SpecContext) {
		gone, err := model.NewUserID()
		Expect(err).NotTo(HaveOccurred())

		_, err = newSessions(data, 12*time.Hour).Issue(ctx, gone)

		Expect(err).To(MatchError(model.ErrUserNotFound))
		Expect(storedTokens(ctx, pool)).To(BeEmpty())
	})
})

var _ = Describe("Logout", func() {
	var (
		pool     *pgxpool.Pool
		sessions *Sessions
		logins   *Logins
		alice    model.UserID
		bob      model.UserID
	)

	BeforeEach(func(ctx SpecContext) {
		var data *store.Store

		pool, data = storetest.New(ctx)
		alice = storetest.AddUser(ctx, data, "alice")
		bob = storetest.AddUser(ctx, data, "bob")

		sessions = newSessions(data, 12*time.Hour)
		logins = NewLogins(nil, sessions)
	})

	issue := func(ctx context.Context, userID model.UserID) secret.String {
		GinkgoHelper()

		token, err := sessions.Issue(ctx, userID)
		Expect(err).NotTo(HaveOccurred())

		return token
	}

	sessionOf := func(ctx context.Context, token secret.String) model.Session {
		GinkgoHelper()

		session, err := sessions.FindByToken(ctx, token)
		Expect(err).NotTo(HaveOccurred())

		return session
	}

	It("revokes only the given session", func(ctx SpecContext) {
		session := sessionOf(ctx, issue(ctx, alice))
		other := issue(ctx, alice)

		err := logins.Logout(ctx, session, false)

		Expect(err).NotTo(HaveOccurred())
		Expect(storedTokens(ctx, pool)).To(ConsistOf(HaveField("TokenHash", Equal(hash(other).Reveal()))))
	})

	It("revokes every token of the user for all sessions, and no other user's", func(ctx SpecContext) {
		session := sessionOf(ctx, issue(ctx, alice))
		issue(ctx, alice)
		bobs := issue(ctx, bob)

		err := logins.Logout(ctx, session, true)

		Expect(err).NotTo(HaveOccurred())
		Expect(storedTokens(ctx, pool)).To(ConsistOf(HaveField("TokenHash", Equal(hash(bobs).Reveal()))))
	})

	DescribeTable("succeeds for a session that is already revoked",
		func(ctx SpecContext, allSessions bool) {
			session := sessionOf(ctx, issue(ctx, alice))
			bobs := issue(ctx, bob)
			Expect(logins.Logout(ctx, session, false)).To(Succeed())

			err := logins.Logout(ctx, session, allSessions)

			Expect(err).NotTo(HaveOccurred())
			Expect(storedTokens(ctx, pool)).To(ConsistOf(HaveField("TokenHash", Equal(hash(bobs).Reveal()))))
		},
		Entry("for one session", false),
		Entry("for all sessions", true),
	)
})

var _ = Describe("FindByToken", func() {
	var (
		pool     *pgxpool.Pool
		sessions *Sessions
		alice    model.UserID
	)

	BeforeEach(func(ctx SpecContext) {
		var data *store.Store

		pool, data = storetest.New(ctx)
		alice = storetest.AddUser(ctx, data, "alice")

		sessions = newSessions(data, 12*time.Hour)
	})

	It("returns the session of the token", func(ctx SpecContext) {
		token, err := sessions.Issue(ctx, alice)
		Expect(err).NotTo(HaveOccurred())

		var id [16]byte
		Expect(pool.QueryRow(ctx, "SELECT id FROM sessions").Scan(&id)).To(Succeed())

		session, err := sessions.FindByToken(ctx, token)

		Expect(err).NotTo(HaveOccurred())
		Expect(session).To(SatisfyAll(
			HaveField("ID", model.SessionID(id)),
			HaveField("UserID", alice),
		))
	})

	It("refuses an unknown token", func(ctx SpecContext) {
		_, err := sessions.Issue(ctx, alice)
		Expect(err).NotTo(HaveOccurred())

		_, err = sessions.FindByToken(ctx, secret.New("unknown"))

		Expect(err).To(MatchError(ErrNotSignedIn))
	})

	It("refuses an expired token", func(ctx SpecContext) {
		expired := secret.New("expired")
		_, err := pool.Exec(ctx, `INSERT INTO sessions (token_hash, user_id, created_at, expires_at)
			VALUES ($1, $2, now(), now() - interval '1 second')`, hash(expired).Reveal(), alice.String())
		Expect(err).NotTo(HaveOccurred())

		_, err = sessions.FindByToken(ctx, expired)

		Expect(err).To(MatchError(ErrNotSignedIn))
	})
})

func newSessions(data *store.Store, lifetime time.Duration) *Sessions {
	return NewSessions(data.Sessions(), data, rnd, clock.NewSystemClock(), lifetime)
}

func storedTokens(ctx context.Context, pool *pgxpool.Pool) []storedToken {
	GinkgoHelper()

	rows, err := pool.Query(ctx, "SELECT token_hash, user_id, expires_at - created_at FROM sessions")
	Expect(err).NotTo(HaveOccurred())

	tokens, err := pgx.CollectRows(rows, pgx.RowToStructByPos[storedToken])
	Expect(err).NotTo(HaveOccurred())

	return tokens
}
