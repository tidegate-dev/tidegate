//go:build integration

package service

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/sync/semaphore"

	"github.com/tidegate-dev/tidegate/authority/internal/db"
	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
	"github.com/tidegate-dev/tidegate/authority/internal/dbtest"
	"github.com/tidegate-dev/tidegate/authority/internal/password"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

var _ = Describe("Authenticate", func() {
	var (
		pool          *pgxpool.Pool
		authenticator *Authenticator
		aliceID       pgtype.UUID
	)

	BeforeEach(func(ctx SpecContext) {
		var err error

		pool, err = db.Open(ctx, dbtest.NewDatabase())
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(pool.Close)
		Expect(db.Migrate(ctx, pool)).To(Succeed())

		Expect(CreateInitialAdmin(ctx, pool, "alice", secret.New(validPassword))).To(Succeed())
		Expect(sqlc.New(pool).CreateUser(ctx, sqlc.CreateUserParams{Name: "bob"})).To(Succeed())

		alice, err := sqlc.New(pool).GetUserByName(ctx, "alice")
		Expect(err).NotTo(HaveOccurred())

		aliceID = alice.ID

		authenticator, err = NewAuthenticator(pool)
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns the user's ID for their password", func(ctx SpecContext) {
		id, err := authenticator.Authenticate(ctx, "alice", secret.New(validPassword))

		Expect(err).NotTo(HaveOccurred())
		Expect(id).To(Equal(aliceID))
	})

	It("ignores the letter case of the name", func(ctx SpecContext) {
		id, err := authenticator.Authenticate(ctx, "ALICE", secret.New(validPassword))

		Expect(err).NotTo(HaveOccurred())
		Expect(id).To(Equal(aliceID))
	})

	DescribeTable("returns ErrWrongCredentials with the reason",
		func(ctx SpecContext, name, plaintext, reason string) {
			_, err := authenticator.Authenticate(ctx, name, secret.New(plaintext))

			Expect(err).To(MatchError(ErrWrongCredentials))
			Expect(err).To(MatchError(ContainSubstring(reason)))
		},
		Entry("wrong password", "alice", "correct horse battery stapler", "wrong password"),
		Entry("unknown user", "carol", validPassword, "unknown user"),
		Entry("no password", "bob", validPassword, "no local password"),
		Entry("invalid name", "al\x00ice", validPassword, "unknown user"),
	)

	It("refuses a password over 1024 bytes", func(ctx SpecContext) {
		_, err := authenticator.Authenticate(ctx, "alice", secret.New(strings.Repeat("a", 1025)))

		Expect(err).To(MatchError(password.ErrTooLong))
	})

	It("refuses a password over 1024 bytes without waiting for a password check", func(ctx SpecContext) {
		authenticator.checks = semaphore.NewWeighted(0)
		authenticator.checkWait = time.Millisecond

		_, err := authenticator.Authenticate(ctx, "alice", secret.New(strings.Repeat("a", 1025)))

		Expect(err).To(MatchError(password.ErrTooLong))
	})

	It("gives up waiting for a password check", func(ctx SpecContext) {
		authenticator.checks = semaphore.NewWeighted(0)
		authenticator.checkWait = time.Millisecond

		_, err := authenticator.Authenticate(ctx, "alice", secret.New(validPassword))

		Expect(err).To(MatchError(ErrBusy))
	})

	It("stops when the context ends", func(ctx SpecContext) {
		canceled, cancel := context.WithCancel(ctx)
		cancel()

		_, err := authenticator.Authenticate(canceled, "alice", secret.New(validPassword))

		Expect(err).To(MatchError(context.Canceled))
	})
})
