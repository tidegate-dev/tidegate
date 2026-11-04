//go:build integration

package login

import (
	"bytes"
	"context"
	"crypto/sha256"
	"log/slog"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/sync/semaphore"

	"github.com/tidegate-dev/tidegate/authority/internal/db"
	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
	"github.com/tidegate-dev/tidegate/authority/internal/dbtest"
	"github.com/tidegate-dev/tidegate/authority/internal/password"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/internal/logging"
)

const alicePassword = "correct horse battery staple"

type storedToken struct {
	TokenHash []byte
	UserID    pgtype.UUID
	Lifetime  time.Duration
}

var _ = Describe("Login", func() {
	var (
		pool    *pgxpool.Pool
		service *Service
		logs    *bytes.Buffer
		aliceID pgtype.UUID
	)

	login := func(ctx context.Context, userName, plaintext string) (string, error) {
		ctx = logging.NewContext(ctx, slog.New(slog.NewTextHandler(logs, nil)))

		res, err := service.Login(ctx, connect.NewRequest(&authorityv1.LoginRequest{
			UserName: userName,
			Password: plaintext,
		}))
		if err != nil {
			return "", err
		}

		return res.Msg.GetToken(), nil
	}

	BeforeEach(func(ctx SpecContext) {
		var err error

		pool, err = db.Open(ctx, dbtest.NewDatabase())
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(pool.Close)
		Expect(db.Migrate(ctx, pool)).To(Succeed())

		hash, err := password.Hash(alicePassword)
		Expect(err).NotTo(HaveOccurred())

		queries := sqlc.New(pool)
		Expect(queries.CreateUser(ctx, sqlc.CreateUserParams{
			Name:         "alice",
			PasswordHash: pgtype.Text{String: hash, Valid: true},
		})).To(Succeed())
		Expect(queries.CreateUser(ctx, sqlc.CreateUserParams{Name: "bob"})).To(Succeed())

		alice, err := queries.GetUserByName(ctx, "alice")
		Expect(err).NotTo(HaveOccurred())

		aliceID = alice.ID

		service, err = NewService(queries, 12*time.Hour)
		Expect(err).NotTo(HaveOccurred())

		logs = &bytes.Buffer{}
	})

	It("returns a token and stores only its hash", func(ctx SpecContext) {
		token, err := login(ctx, "alice", alicePassword)

		Expect(err).NotTo(HaveOccurred())
		Expect(token).To(MatchRegexp(`^[A-Za-z0-9_-]{43}$`))

		hash := sha256.Sum256([]byte(token))
		Expect(storedTokens(ctx, pool)).To(ConsistOf(SatisfyAll(
			HaveField("TokenHash", Equal(hash[:])),
			HaveField("UserID", aliceID),
			HaveField("Lifetime", 12*time.Hour),
		)))
	})

	It("returns a new token for each login", func(ctx SpecContext) {
		first, err := login(ctx, "alice", alicePassword)
		Expect(err).NotTo(HaveOccurred())

		second, err := login(ctx, "alice", alicePassword)
		Expect(err).NotTo(HaveOccurred())

		Expect(second).NotTo(Equal(first))
		Expect(storedTokens(ctx, pool)).To(HaveLen(2))
	})

	It("ignores the letter case of the user name", func(ctx SpecContext) {
		_, err := login(ctx, "ALICE", alicePassword)

		Expect(err).NotTo(HaveOccurred())
	})

	It("stores the expiry one token lifetime after the login", func(ctx SpecContext) {
		var err error

		service, err = NewService(sqlc.New(pool), time.Second)
		Expect(err).NotTo(HaveOccurred())

		_, err = login(ctx, "alice", alicePassword)

		Expect(err).NotTo(HaveOccurred())
		Expect(storedTokens(ctx, pool)).To(ConsistOf(HaveField("Lifetime", time.Second)))
	})

	It("deletes the user's expired tokens", func(ctx SpecContext) {
		_, err := pool.Exec(ctx, `INSERT INTO login_tokens (token_hash, user_id, expires_at)
			VALUES (sha256('expired'), $1, now() - interval '1 second')`, aliceID)
		Expect(err).NotTo(HaveOccurred())

		token, err := login(ctx, "alice", alicePassword)

		Expect(err).NotTo(HaveOccurred())

		hash := sha256.Sum256([]byte(token))
		Expect(storedTokens(ctx, pool)).To(ConsistOf(HaveField("TokenHash", Equal(hash[:]))))
	})

	It("gives a wrong password, an unknown user and a user without a password the same error", func(ctx SpecContext) {
		_, wrongPassword := login(ctx, "alice", "correct horse battery stapler")
		_, unknownUser := login(ctx, "carol", alicePassword)
		_, noPassword := login(ctx, "bob", alicePassword)

		Expect(connect.CodeOf(wrongPassword)).To(Equal(connect.CodeUnauthenticated))
		Expect(unknownUser).To(Equal(wrongPassword))
		Expect(noPassword).To(Equal(wrongPassword))
		Expect(storedTokens(ctx, pool)).To(BeEmpty())
	})

	It("refuses a password over 1024 bytes", func(ctx SpecContext) {
		_, err := login(ctx, "alice", strings.Repeat("a", 1025))

		Expect(connect.CodeOf(err)).To(Equal(connect.CodeInvalidArgument))
		Expect(storedTokens(ctx, pool)).To(BeEmpty())
	})

	It("logs a successful login with the user name and without the password or token", func(ctx SpecContext) {
		token, err := login(ctx, "alice", alicePassword)

		Expect(err).NotTo(HaveOccurred())
		Expect(logs.String()).To(SatisfyAll(
			ContainSubstring("user=alice"),
			Not(ContainSubstring(alicePassword)),
			Not(ContainSubstring(token)),
		))
	})

	It("gives up waiting for a password check and logs it", func(ctx SpecContext) {
		service.checks = semaphore.NewWeighted(0)
		service.checkWait = time.Millisecond

		_, err := login(ctx, "alice", alicePassword)

		Expect(connect.CodeOf(err)).To(Equal(connect.CodeUnavailable))
		Expect(logs.String()).To(ContainSubstring("user=alice"))
		Expect(storedTokens(ctx, pool)).To(BeEmpty())
	})

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

	DescribeTable("logs a failed login with the user name and without the password",
		func(ctx SpecContext, userName, plaintext string) {
			_, err := login(ctx, userName, plaintext)

			Expect(err).To(HaveOccurred())
			Expect(logs.String()).To(SatisfyAll(
				ContainSubstring("user="+userName),
				Not(ContainSubstring(plaintext)),
			))
		},
		Entry("wrong password", "alice", "correct horse battery stapler"),
		Entry("unknown user", "carol", alicePassword),
		Entry("no password", "bob", alicePassword),
		Entry("password too long", "alice", strings.Repeat("a", 1025)),
	)
})

func storedTokens(ctx context.Context, pool *pgxpool.Pool) []storedToken {
	GinkgoHelper()

	rows, err := pool.Query(ctx, "SELECT token_hash, user_id, expires_at - created_at FROM login_tokens")
	Expect(err).NotTo(HaveOccurred())

	tokens, err := pgx.CollectRows(rows, pgx.RowToStructByPos[storedToken])
	Expect(err).NotTo(HaveOccurred())

	return tokens
}
