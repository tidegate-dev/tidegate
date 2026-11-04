//go:build integration

package service

import (
	"context"
	"crypto/sha256"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/db"
	"github.com/tidegate-dev/tidegate/authority/internal/dbtest"
)

type storedToken struct {
	TokenHash []byte
	UserID    pgtype.UUID
	Lifetime  time.Duration
}

var _ = Describe("Issue", func() {
	var (
		pool   *pgxpool.Pool
		userID pgtype.UUID
	)

	BeforeEach(func(ctx SpecContext) {
		var err error

		pool, err = db.Open(ctx, dbtest.NewDatabase())
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(pool.Close)
		Expect(db.Migrate(ctx, pool)).To(Succeed())

		Expect(pool.QueryRow(ctx, "INSERT INTO users (name) VALUES ('alice') RETURNING id").Scan(&userID)).To(Succeed())
	})

	It("returns a token and stores only its hash", func(ctx SpecContext) {
		token, err := NewLoginTokens(pool, 12*time.Hour).Issue(ctx, userID)

		Expect(err).NotTo(HaveOccurred())
		Expect(token.Reveal()).To(MatchRegexp(`^[A-Za-z0-9_-]{43}$`))

		hash := sha256.Sum256([]byte(token.Reveal()))
		Expect(storedTokens(ctx, pool)).To(ConsistOf(SatisfyAll(
			HaveField("TokenHash", Equal(hash[:])),
			HaveField("UserID", userID),
			HaveField("Lifetime", 12*time.Hour),
		)))
	})

	It("returns a new token each time", func(ctx SpecContext) {
		tokens := NewLoginTokens(pool, 12*time.Hour)

		first, err := tokens.Issue(ctx, userID)
		Expect(err).NotTo(HaveOccurred())

		second, err := tokens.Issue(ctx, userID)
		Expect(err).NotTo(HaveOccurred())

		Expect(second).NotTo(Equal(first))
		Expect(storedTokens(ctx, pool)).To(HaveLen(2))
	})

	It("stores the expiry one lifetime after issuing the token", func(ctx SpecContext) {
		_, err := NewLoginTokens(pool, time.Second).Issue(ctx, userID)

		Expect(err).NotTo(HaveOccurred())
		Expect(storedTokens(ctx, pool)).To(ConsistOf(HaveField("Lifetime", time.Second)))
	})

	It("deletes the user's expired tokens", func(ctx SpecContext) {
		_, err := pool.Exec(ctx, `INSERT INTO login_tokens (token_hash, user_id, expires_at)
			VALUES (sha256('expired'), $1, now() - interval '1 second')`, userID)
		Expect(err).NotTo(HaveOccurred())

		token, err := NewLoginTokens(pool, 12*time.Hour).Issue(ctx, userID)

		Expect(err).NotTo(HaveOccurred())

		hash := sha256.Sum256([]byte(token.Reveal()))
		Expect(storedTokens(ctx, pool)).To(ConsistOf(HaveField("TokenHash", Equal(hash[:]))))
	})
})

func storedTokens(ctx context.Context, pool *pgxpool.Pool) []storedToken {
	GinkgoHelper()

	rows, err := pool.Query(ctx, "SELECT token_hash, user_id, expires_at - created_at FROM login_tokens")
	Expect(err).NotTo(HaveOccurred())

	tokens, err := pgx.CollectRows(rows, pgx.RowToStructByPos[storedToken])
	Expect(err).NotTo(HaveOccurred())

	return tokens
}
