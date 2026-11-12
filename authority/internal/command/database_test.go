//go:build integration

package command

import (
	"context"
	"net"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/The127/go-random"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/config"
	"github.com/tidegate-dev/tidegate/authority/internal/db"
	"github.com/tidegate-dev/tidegate/authority/internal/dbtest"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1/authorityv1connect"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

var _ = Describe("run", func() {
	When("a query hangs", func() {
		var authority runningAuthority

		BeforeEach(func(ctx SpecContext) {
			url := dbtest.NewDatabase()
			Expect(db.MigrateAt(ctx, url)).To(Succeed())
			authority = serveAuthority(ctx, url)
			token := signIn(ctx, url)

			// The lock makes the authority's write wait until the spec ends,
			// like a query to a Postgres that doesn't answer.
			holder := connectToDatabase(ctx, url)
			lockDummies(ctx, holder)

			go writeDummy(authority.address, token)

			Eventually(ctx, countSessionsWaitingForALock).WithArguments(holder).Should(Equal(1), "the write never waited for the lock")
		}, NodeTimeout(30*time.Second))

		It("stops", func(ctx SpecContext) {
			authority.stop()

			Eventually(ctx, authority.runErrorChan).WithTimeout(5*time.Second).Should(Receive(), "the authority didn't stop")
		}, SpecTimeout(30*time.Second))
	})
})

// runningAuthority is an authority that serves in the background.
type runningAuthority struct {
	address      string
	stop         context.CancelFunc
	runErrorChan chan error
}

// serveAuthority runs the authority on the database at url and returns once
// its API listens. It waits only 100 milliseconds for running requests when
// it stops, so the spec doesn't wait the full shutdown timeout.
func serveAuthority(ctx context.Context, url string) runningAuthority {
	GinkgoHelper()

	runCtx, stop := context.WithCancel(context.Background())
	DeferCleanup(stop)

	authority := runningAuthority{
		address:      freeAddress(GinkgoTB()),
		stop:         stop,
		runErrorChan: make(chan error, 1),
	}

	go func() {
		authority.runErrorChan <- run(runCtx, config.Config{
			Server:   config.Server{ListenAddress: authority.address, TLS: config.TLS{Disabled: true}},
			Metrics:  config.Metrics{ListenAddress: "127.0.0.1:0"},
			Database: config.Database{URL: url},
		}, 100*time.Millisecond)
	}()

	Eventually(ctx, listening).WithArguments(authority.address).Should(Succeed())

	return authority
}

// lockDummies locks the dummies table in a transaction that stays open
// until the connection closes.
func lockDummies(ctx context.Context, conn *pgx.Conn) {
	GinkgoHelper()

	_, err := conn.Exec(ctx, "BEGIN; LOCK TABLE dummies IN ACCESS EXCLUSIVE MODE")
	Expect(err).NotTo(HaveOccurred())
}

func countSessionsWaitingForALock(ctx context.Context, conn *pgx.Conn) (int, error) {
	var sessions int

	err := conn.QueryRow(ctx,
		"SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock'",
	).Scan(&sessions)

	return sessions, err
}

func listening(address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}

	return conn.Close()
}

// writeDummy sends a write that only ends with the authority. A request
// whose client gives up would cancel its query without the shutdown.
func writeDummy(address string, token secret.String) {
	client := authorityv1connect.NewDummyServiceClient(http.DefaultClient, "http://"+address)
	request := connect.NewRequest(&authorityv1.WriteRequest{Text: "waits"})
	request.Header().Set("Authorization", "Bearer "+token.Reveal())

	_, _ = client.Write(context.Background(), request)
}

func signIn(ctx context.Context, url string) secret.String {
	GinkgoHelper()

	pool, err := db.Open(ctx, url)
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(pool.Close)

	var userID pgtype.UUID
	Expect(pool.QueryRow(ctx, "INSERT INTO users (name) VALUES ('alice') RETURNING id").Scan(&userID)).To(Succeed())

	token, err := service.NewSessions(pool, random.NewCryptoRandom(), time.Hour).Issue(ctx, userID)
	Expect(err).NotTo(HaveOccurred())

	return token
}

func connectToDatabase(ctx context.Context, url string) *pgx.Conn {
	GinkgoHelper()

	conn, err := pgx.Connect(ctx, url)
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() {
		_ = conn.Close(context.Background())
	})

	return conn
}
