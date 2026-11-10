//go:build integration

package main

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
	"github.com/pressly/goose/v3/lock"

	"github.com/tidegate-dev/tidegate/authority/internal/config"
	"github.com/tidegate-dev/tidegate/authority/internal/db"
	"github.com/tidegate-dev/tidegate/authority/internal/dbtest"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1/authorityv1connect"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

var _ = Describe("openDatabase", func() {
	It("refuses an empty database", func(ctx SpecContext) {
		url := dbtest.NewDatabase()

		_, err := openDatabase(ctx, url)

		Expect(err).To(MatchError(db.ErrSchemaTooOld))
		Expect(err).To(MatchError(ContainSubstring("Run authority migrate first")))
	})

	It("opens a migrated database", func(ctx SpecContext) {
		url := dbtest.NewDatabase()
		Expect(migrate(ctx, config.Database{URL: url})).To(Succeed())

		pool, err := openDatabase(ctx, url)

		Expect(err).NotTo(HaveOccurred())
		pool.Close()
	})
})

var _ = Describe("migrate", func() {
	It("migrates once when two authorities wait for the lock", func(ctx SpecContext) {
		url := dbtest.NewDatabase()
		holder := connectToDatabase(ctx, url)
		takeMigrationLock(ctx, holder)

		firstMigrationErrorChan := startMigration(ctx, url)
		secondMigrationErrorChan := startMigration(ctx, url)
		Eventually(ctx, countSessionsTryingMigrationLock).WithArguments(holder).WithTimeout(30*time.Second).
			Should(Equal(2), "the authorities never waited for goose's migration lock")

		releaseMigrationLock(ctx, holder)

		Expect(<-firstMigrationErrorChan).To(Succeed())
		Expect(<-secondMigrationErrorChan).To(Succeed())
		Expect(maxTimesAMigrationWasApplied(ctx, holder)).To(Equal(1))
	}, SpecTimeout(time.Minute))
})

var _ = Describe("run", func() {
	When("a query hangs", func() {
		var authority runningAuthority

		BeforeEach(func(ctx SpecContext) {
			url := dbtest.NewDatabase()
			Expect(migrate(ctx, config.Database{URL: url})).To(Succeed())
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

// startMigration migrates the database in the background and sends
// migrate's error on the returned channel.
func startMigration(ctx context.Context, url string) <-chan error {
	migrationErrorChan := make(chan error, 1)

	go func() {
		migrationErrorChan <- migrate(ctx, config.Database{URL: url})
	}()

	return migrationErrorChan
}

func takeMigrationLock(ctx context.Context, conn *pgx.Conn) {
	GinkgoHelper()

	_, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", lock.DefaultLockID)
	Expect(err).NotTo(HaveOccurred())
}

func releaseMigrationLock(ctx context.Context, conn *pgx.Conn) {
	GinkgoHelper()

	_, err := conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", lock.DefaultLockID)
	Expect(err).NotTo(HaveOccurred())
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

	token, err := service.NewLoginTokens(pool, random.NewCryptoRandom(), time.Hour).Issue(ctx, userID)
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

func countSessionsTryingMigrationLock(ctx context.Context, conn *pgx.Conn) (int, error) {
	var sessions int

	// Between goose's attempts, pg_stat_activity shows the last attempt as
	// the session's query.
	err := conn.QueryRow(ctx,
		"SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND pid <> pg_backend_pid() AND query LIKE '%pg_try_advisory_lock%'",
	).Scan(&sessions)

	return sessions, err
}

// maxTimesAMigrationWasApplied counts goose's rows for each migration and
// returns the highest count.
func maxTimesAMigrationWasApplied(ctx context.Context, conn *pgx.Conn) int {
	GinkgoHelper()

	var count int

	err := conn.QueryRow(ctx,
		"SELECT max(applied) FROM (SELECT count(*) AS applied FROM goose_db_version WHERE version_id > 0 GROUP BY version_id) AS migrations",
	).Scan(&count)
	Expect(err).NotTo(HaveOccurred())

	return count
}
