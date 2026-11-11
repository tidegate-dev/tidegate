//go:build integration

package api

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/The127/go-random/randomtest"
	"github.com/jackc/pgx/v5/pgtype"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/db"
	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
	"github.com/tidegate-dev/tidegate/authority/internal/dbtest"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/internal/logging"
	"github.com/tidegate-dev/tidegate/internal/secret"
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

		Expect(service.CreateInitialAdmin(ctx, pool, randomtest.NewMock(), "alice", secret.New(alicePassword))).To(Succeed())
		Expect(sqlc.New(pool).CreateUser(ctx, sqlc.CreateUserParams{Name: "bob"})).To(Succeed())

		authenticator, err := service.NewAuthenticator(pool, randomtest.NewMock())
		Expect(err).NotTo(HaveOccurred())

		handler = NewLoginService(service.NewLogins(authenticator, service.NewSessions(pool, randomtest.NewMock(), 12*time.Hour)))
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

var _ = Describe("Logout", func() {
	var (
		handler *LoginService
		tokens  *service.Sessions
		token   secret.String
		session service.Session
	)

	BeforeEach(func(ctx SpecContext) {
		pool, err := db.Open(ctx, dbtest.NewDatabase())
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(pool.Close)
		Expect(db.Migrate(ctx, pool)).To(Succeed())

		authenticator, err := service.NewAuthenticator(pool, randomtest.NewMock())
		Expect(err).NotTo(HaveOccurred())

		tokens = service.NewSessions(pool, randomtest.NewMock(), 12*time.Hour)
		handler = NewLoginService(service.NewLogins(authenticator, tokens))

		var alice pgtype.UUID
		Expect(pool.QueryRow(ctx, "INSERT INTO users (name) VALUES ('alice') RETURNING id").Scan(&alice)).To(Succeed())

		token, err = tokens.Issue(ctx, alice)
		Expect(err).NotTo(HaveOccurred())

		session, err = tokens.Session(ctx, token)
		Expect(err).NotTo(HaveOccurred())
	})

	It("revokes the session of the call", func(ctx SpecContext) {
		signedIn := context.WithValue(ctx, sessionKey{}, session)

		_, err := handler.Logout(signedIn, connect.NewRequest(&authorityv1.LogoutRequest{}))

		Expect(err).NotTo(HaveOccurred())

		_, err = tokens.Session(ctx, token)
		Expect(err).To(MatchError(service.ErrNotSignedIn))
	})

	It("fails without revoking anything if the call has no session", func(ctx SpecContext) {
		_, err := handler.Logout(ctx, connect.NewRequest(&authorityv1.LogoutRequest{AllSessions: true}))

		Expect(err).To(MatchError(errNoSession))

		_, err = tokens.Session(ctx, token)
		Expect(err).NotTo(HaveOccurred())
	})
})
