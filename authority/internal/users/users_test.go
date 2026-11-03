//go:build integration

package users

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/db"
	"github.com/tidegate-dev/tidegate/authority/internal/dbtest"
)

const validPassword = "correct horse battery staple"

var _ = Describe("CreateInitialAdmin", func() {
	var pool *pgxpool.Pool

	BeforeEach(func(ctx SpecContext) {
		var err error

		pool, err = db.Open(ctx, dbtest.NewDatabase())
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(pool.Close)
		Expect(db.Migrate(ctx, pool)).To(Succeed())
	})

	It("creates the user with an argon2id hash of the password", func(ctx SpecContext) {
		err := CreateInitialAdmin(ctx, pool, "alice", validPassword)

		Expect(err).NotTo(HaveOccurred())
		Expect(storedUsers(ctx, pool)).To(ConsistOf(SatisfyAll(
			HaveField("Name", "alice"),
			HaveField("PasswordHash", HavePrefix("$argon2id$v=19$m=65536,t=3,p=4$")),
		)))
	})

	It("refuses when a user already exists", func(ctx SpecContext) {
		Expect(CreateInitialAdmin(ctx, pool, "alice", validPassword)).To(Succeed())

		err := CreateInitialAdmin(ctx, pool, "bob", validPassword)

		Expect(err).To(MatchError(ErrUserExists))
		Expect(storedUsers(ctx, pool)).To(HaveLen(1))
	})

	DescribeTable("refuses a name that's empty or has whitespace around it",
		func(ctx SpecContext, name string) {
			err := CreateInitialAdmin(ctx, pool, name, validPassword)

			Expect(err).To(MatchError(ErrInvalidName))
			Expect(storedUsers(ctx, pool)).To(BeEmpty())
		},
		Entry("empty", ""),
		Entry("only whitespace", " "),
		Entry("leading space", " alice"),
		Entry("trailing newline", "alice\n"),
	)

	It("creates one user when two runs wait for the lock", func(ctx SpecContext) {
		holder, err := pool.Acquire(ctx)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(holder.Release)
		_, err = holder.Exec(ctx, "SELECT pg_advisory_lock($1)", initialAdminLockID)
		Expect(err).NotTo(HaveOccurred())

		firstCreationErrorChan := startCreateInitialAdmin(ctx, pool, "alice")
		secondCreationErrorChan := startCreateInitialAdmin(ctx, pool, "bob")
		Eventually(ctx, countSessionsWaitingForInitialAdminLock).WithArguments(pool).WithTimeout(30*time.Second).
			Should(Equal(2), "the runs never waited for the lock")

		_, err = holder.Exec(ctx, "SELECT pg_advisory_unlock($1)", initialAdminLockID)
		Expect(err).NotTo(HaveOccurred())

		Expect([]error{<-firstCreationErrorChan, <-secondCreationErrorChan}).To(ConsistOf(Succeed(), MatchError(ErrUserExists)))
		Expect(storedUsers(ctx, pool)).To(HaveLen(1))
	}, SpecTimeout(time.Minute))
})

type storedUser struct {
	Name         string
	PasswordHash string
}

func storedUsers(ctx context.Context, pool *pgxpool.Pool) []storedUser {
	GinkgoHelper()

	rows, err := pool.Query(ctx, "SELECT name, password_hash FROM users")
	Expect(err).NotTo(HaveOccurred())

	users, err := pgx.CollectRows(rows, pgx.RowToStructByPos[storedUser])
	Expect(err).NotTo(HaveOccurred())

	return users
}

func startCreateInitialAdmin(ctx context.Context, pool *pgxpool.Pool, name string) <-chan error {
	creationErrorChan := make(chan error, 1)

	go func() {
		creationErrorChan <- CreateInitialAdmin(ctx, pool, name, validPassword)
	}()

	return creationErrorChan
}

func countSessionsWaitingForInitialAdminLock(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var sessions int

	// pg_locks splits a bigint advisory key into objid (low 32 bits) and
	// classid (high 32 bits).
	err := pool.QueryRow(ctx,
		"SELECT count(*) FROM pg_locks WHERE locktype = 'advisory' AND NOT granted AND ((classid::bigint << 32) | objid::bigint) = $1",
		initialAdminLockID,
	).Scan(&sessions)

	return sessions, err
}
