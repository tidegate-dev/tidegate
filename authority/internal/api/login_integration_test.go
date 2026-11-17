//go:build integration

package api

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/The127/go-clock"
	"github.com/The127/go-random/randomtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
	"github.com/tidegate-dev/tidegate/authority/internal/store"
	"github.com/tidegate-dev/tidegate/authority/internal/storetest"
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
		_, data := storetest.New(ctx)

		Expect(service.NewUsers(data.Users(), data, randomtest.NewMock()).CreateInitialAdmin(ctx, "alice", secret.New(alicePassword))).To(Succeed())
		storetest.AddUser(ctx, data, "bob")

		authenticator, err := service.NewAuthenticator(data.Users(), randomtest.NewMock())
		Expect(err).NotTo(HaveOccurred())

		handler = NewLoginService(service.NewLogins(authenticator, newSessions(data)))
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
		handler  *LoginService
		sessions *service.Sessions
		token    secret.String
		session  model.Session
	)

	BeforeEach(func(ctx SpecContext) {
		_, data := storetest.New(ctx)

		authenticator, err := service.NewAuthenticator(data.Users(), randomtest.NewMock())
		Expect(err).NotTo(HaveOccurred())

		sessions = newSessions(data)
		handler = NewLoginService(service.NewLogins(authenticator, sessions))

		token, err = sessions.Issue(ctx, storetest.AddUser(ctx, data, "alice"))
		Expect(err).NotTo(HaveOccurred())

		session, err = sessions.Session(ctx, token)
		Expect(err).NotTo(HaveOccurred())
	})

	It("revokes the session of the call", func(ctx SpecContext) {
		signedIn := context.WithValue(ctx, sessionKey{}, session)

		_, err := handler.Logout(signedIn, connect.NewRequest(&authorityv1.LogoutRequest{}))

		Expect(err).NotTo(HaveOccurred())

		_, err = sessions.Session(ctx, token)
		Expect(err).To(MatchError(service.ErrNotSignedIn))
	})

	It("fails without revoking anything if the call has no session", func(ctx SpecContext) {
		_, err := handler.Logout(ctx, connect.NewRequest(&authorityv1.LogoutRequest{AllSessions: true}))

		Expect(err).To(MatchError(errNoSession))

		_, err = sessions.Session(ctx, token)
		Expect(err).NotTo(HaveOccurred())
	})
})

func newSessions(data *store.Store) *service.Sessions {
	return service.NewSessions(data.Sessions(), data, randomtest.NewMock(), clock.NewSystemClock(), 12*time.Hour)
}
