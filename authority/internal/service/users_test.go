//go:build integration

package service

import (
	"context"

	"github.com/The127/go-random/randomtest"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/phemera-dev/phemera/authority/internal/model"
	"github.com/phemera-dev/phemera/authority/internal/store"
	"github.com/phemera-dev/phemera/authority/internal/storetest"
	"github.com/phemera-dev/phemera/internal/secret"
)

const validPassword = "correct horse battery staple"

var rnd = randomtest.NewMock()

var _ = Describe("Users", func() {
	var (
		pool  *pgxpool.Pool
		users *Users
	)

	BeforeEach(func(ctx SpecContext) {
		var data *store.Store

		pool, data = storetest.New(ctx)
		users = NewUsers(data.Users(), data, rnd)
	})

	Describe("CreateInitialAdmin", func() {
		It("creates the user with an argon2id hash of the password", func(ctx SpecContext) {
			err := users.CreateInitialAdmin(ctx, "alice", secret.New(validPassword))

			Expect(err).NotTo(HaveOccurred())
			Expect(storedUsers(ctx, pool)).To(ConsistOf(SatisfyAll(
				HaveField("Name", "alice"),
				HaveField("PasswordHash", HavePrefix("$argon2id$v=19$m=65536,t=3,p=4$")),
			)))
		})

		It("refuses when a user already exists", func(ctx SpecContext) {
			Expect(users.CreateInitialAdmin(ctx, "alice", secret.New(validPassword))).To(Succeed())

			err := users.CreateInitialAdmin(ctx, "bob", secret.New(validPassword))

			Expect(err).To(MatchError(model.ErrUsersExist))
			Expect(storedUsers(ctx, pool)).To(HaveLen(1))
		})

		DescribeTable("refuses an invalid name",
			func(ctx SpecContext, name string) {
				err := users.CreateInitialAdmin(ctx, name, secret.New(validPassword))

				Expect(err).To(MatchError(ErrInvalidName))
				Expect(storedUsers(ctx, pool)).To(BeEmpty())
			},
			Entry("empty", ""),
			Entry("only whitespace", " "),
			Entry("leading space", " alice"),
			Entry("trailing newline", "alice\n"),
			Entry("NUL", "al\x00ice"),
		)
	})
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
