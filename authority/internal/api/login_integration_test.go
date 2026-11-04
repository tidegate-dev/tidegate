//go:build integration

package api

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"time"

	"connectrpc.com/connect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/db"
	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
	"github.com/tidegate-dev/tidegate/authority/internal/dbtest"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/internal/logging"
)

const alicePassword = "correct horse battery staple"

var _ = Describe("Login", func() {
	var (
		handler *LoginService
		logs    *bytes.Buffer
	)

	login := func(ctx context.Context, userName, plaintext string) (string, error) {
		ctx = logging.NewContext(ctx, slog.New(slog.NewTextHandler(logs, nil)))

		res, err := handler.Login(ctx, connect.NewRequest(&authorityv1.LoginRequest{
			UserName: userName,
			Password: plaintext,
		}))
		if err != nil {
			return "", err
		}

		return res.Msg.GetToken(), nil
	}

	BeforeEach(func(ctx SpecContext) {
		pool, err := db.Open(ctx, dbtest.NewDatabase())
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(pool.Close)
		Expect(db.Migrate(ctx, pool)).To(Succeed())

		Expect(service.CreateInitialAdmin(ctx, pool, "alice", alicePassword)).To(Succeed())
		Expect(sqlc.New(pool).CreateUser(ctx, sqlc.CreateUserParams{Name: "bob"})).To(Succeed())

		authenticator, err := service.NewAuthenticator(pool)
		Expect(err).NotTo(HaveOccurred())

		handler = NewLoginService(service.NewLogins(authenticator, service.NewLoginTokens(pool, 12*time.Hour)))
		logs = &bytes.Buffer{}
	})

	It("returns a token for the right password", func(ctx SpecContext) {
		token, err := login(ctx, "alice", alicePassword)

		Expect(err).NotTo(HaveOccurred())
		Expect(token).NotTo(BeEmpty())
	})

	It("gives a wrong password, an unknown user and a user without a password the same error", func(ctx SpecContext) {
		_, wrongPassword := login(ctx, "alice", "correct horse battery stapler")
		_, unknownUser := login(ctx, "carol", alicePassword)
		_, noPassword := login(ctx, "bob", alicePassword)

		Expect(connect.CodeOf(wrongPassword)).To(Equal(connect.CodeUnauthenticated))
		Expect(unknownUser).To(Equal(wrongPassword))
		Expect(noPassword).To(Equal(wrongPassword))
	})

	It("logs a successful login with the user name and without the password or token", func(ctx SpecContext) {
		token, err := login(ctx, "alice", alicePassword)

		Expect(err).NotTo(HaveOccurred())
		Expect(logs.String()).To(SatisfyAll(
			ContainSubstring("login succeeded"),
			ContainSubstring("user=alice"),
			Not(ContainSubstring(alicePassword)),
			Not(ContainSubstring(token)),
		))
	})

	DescribeTable("logs a failed login with the user name and reason, without the password",
		func(ctx SpecContext, userName, plaintext, reason string) {
			_, err := login(ctx, userName, plaintext)

			Expect(err).To(HaveOccurred())
			Expect(logs.String()).To(SatisfyAll(
				ContainSubstring("login failed"),
				ContainSubstring("user="+userName),
				ContainSubstring(reason),
				Not(ContainSubstring(plaintext)),
			))
		},
		Entry("wrong password", "alice", "correct horse battery stapler", "wrong password"),
		Entry("unknown user", "carol", alicePassword, "unknown user"),
		Entry("no password", "bob", alicePassword, "no local password"),
		Entry("password too long", "alice", strings.Repeat("a", 1025), "at most 1024 bytes"),
	)

	It("logs a login whose client gave up", func(ctx SpecContext) {
		canceled, cancel := context.WithCancel(ctx)
		cancel()

		_, err := login(canceled, "alice", alicePassword)

		Expect(err).To(MatchError(context.Canceled))
		Expect(logs.String()).To(SatisfyAll(
			ContainSubstring("login failed"),
			ContainSubstring("user=alice"),
		))
	})
})
