//go:build integration

package main

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/config"
	"github.com/tidegate-dev/tidegate/authority/internal/dbtest"
	"github.com/tidegate-dev/tidegate/authority/internal/password"
	"github.com/tidegate-dev/tidegate/authority/internal/users"
)

var _ = Describe("authority create-initial-admin", func() {
	var url string

	BeforeEach(func(ctx SpecContext) {
		url = dbtest.NewDatabase()
		GinkgoT().Setenv("TIDEGATE_AUTHORITY__DATABASE__URL", url)
		Expect(migrate(ctx, config.Database{URL: url})).To(Succeed())
	})

	It("creates the user with the password from stdin", func(ctx SpecContext) {
		err := runCreateInitialAdmin(ctx, "alice", "correct horse battery staple\n")

		Expect(err).NotTo(HaveOccurred())
		Expect(userNames(ctx, url)).To(ConsistOf("alice"))
	})

	It("doesn't count the line ending as part of the password", func(ctx SpecContext) {
		err := runCreateInitialAdmin(ctx, "alice", "fourteen chars\n")

		Expect(err).To(MatchError(password.ErrTooShort))
		Expect(userNames(ctx, url)).To(BeEmpty())
	})

	It("refuses an invalid name before it reads a password", func(ctx SpecContext) {
		command := newCommand()
		command.Reader = unreadable{}

		err := command.Run(ctx, []string{"authority", "create-initial-admin", " alice"})

		Expect(err).To(MatchError(users.ErrInvalidName))
	})

	It("refuses before it reads a password when a user exists", func(ctx SpecContext) {
		Expect(runCreateInitialAdmin(ctx, "alice", "correct horse battery staple\n")).To(Succeed())

		err := runCreateInitialAdmin(ctx, "bob", "")

		Expect(err).To(MatchError(users.ErrUserExists))
	})
})

func runCreateInitialAdmin(ctx context.Context, name, stdin string) error {
	command := newCommand()
	command.Reader = strings.NewReader(stdin)

	return command.Run(ctx, []string{"authority", "create-initial-admin", name})
}

// unreadable is a stdin that fails every read.
type unreadable struct{}

func (unreadable) Read([]byte) (int, error) {
	return 0, errors.New("the command read stdin")
}

func userNames(ctx context.Context, url string) []string {
	GinkgoHelper()

	conn := connectToDatabase(ctx, url)

	rows, err := conn.Query(ctx, "SELECT name FROM users")
	Expect(err).NotTo(HaveOccurred())

	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	Expect(err).NotTo(HaveOccurred())

	return names
}
