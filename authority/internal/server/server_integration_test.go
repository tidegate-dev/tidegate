//go:build integration

package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/The127/go-clock"
	"github.com/The127/go-random"
	"github.com/jackc/pgx/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/phemera-dev/phemera/authority/internal/config"
	"github.com/phemera-dev/phemera/authority/internal/db"
	"github.com/phemera-dev/phemera/authority/internal/dbtest"
	"github.com/phemera-dev/phemera/authority/internal/service"
	"github.com/phemera-dev/phemera/authority/internal/store"
	"github.com/phemera-dev/phemera/authority/internal/storetest"
	authorityv1 "github.com/phemera-dev/phemera/gen/phemera/authority/v1"
	"github.com/phemera-dev/phemera/gen/phemera/authority/v1/authorityv1connect"
	"github.com/phemera-dev/phemera/internal/secret"
)

var _ = Describe("run", func() {
	When("a query hangs", func() {
		var authority runningAuthority

		BeforeEach(func(ctx SpecContext) {
			url := dbtest.NewDatabase()
			Expect(MigrateDatabase(ctx, url)).To(Succeed())
			authority = serveAuthority(ctx, url)
			token := signIn(ctx, url)

			// The lock makes the authority's write wait until the spec ends,
			// like a query to a Postgres that doesn't answer.
			holder := dbtest.Connect(ctx, url)
			lockProjects(ctx, holder)

			go createProject(authority.externalAddress, token)

			Eventually(ctx, countSessionsWaitingForALock).WithArguments(holder).Should(Equal(1), "the write never waited for the lock")
		}, NodeTimeout(30*time.Second))

		It("stops", func(ctx SpecContext) {
			authority.stop()

			Eventually(ctx, authority.runErrorChan).WithTimeout(5*time.Second).Should(Receive(), "the authority didn't stop")
		}, SpecTimeout(30*time.Second))
	})

	When("one of the external API's procedures is called on the internal API's address", func() {
		var authority runningAuthority

		BeforeEach(func(ctx SpecContext) {
			url := dbtest.NewDatabase()
			Expect(MigrateDatabase(ctx, url)).To(Succeed())
			authority = serveAuthority(ctx, url)
		}, NodeTimeout(30*time.Second))

		It("isn't found, over TLS with the internal API's own certificate", func(ctx SpecContext) {
			client := clientTrusting(authority.internalCertFile)
			procedures := []string{
				authorityv1connect.LoginServiceLoginProcedure,
				authorityv1connect.ProjectServiceListProjectsProcedure,
			}

			for _, procedure := range procedures {
				url := "https://" + authority.internalAddress + procedure

				request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader("{}"))
				Expect(err).NotTo(HaveOccurred())
				request.Header.Set("Content-Type", "application/json")

				response, err := client.Do(request)
				Expect(err).NotTo(HaveOccurred())
				Expect(response.Body.Close()).To(Succeed())

				Expect(response.StatusCode).To(Equal(http.StatusNotFound), procedure)
			}
		}, SpecTimeout(30*time.Second))
	})

	When("the database answers", func() {
		var authority runningAuthority

		BeforeEach(func(ctx SpecContext) {
			url := dbtest.NewDatabase()
			Expect(MigrateDatabase(ctx, url)).To(Succeed())
			authority = serveAuthority(ctx, url)
		}, NodeTimeout(30*time.Second))

		It("answers its readiness check with 200 OK", func(ctx SpecContext) {
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+authority.metricsAddress+"/readyz", nil)
			Expect(err).NotTo(HaveOccurred())

			response, err := http.DefaultClient.Do(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Body.Close()).To(Succeed())

			Expect(response.StatusCode).To(Equal(http.StatusOK))
		}, SpecTimeout(30*time.Second))
	})

	DescribeTable("names the server whose listen address is taken",
		func(ctx SpecContext, serverName string, setListenAddress func(cfg *config.Config, address string)) {
			url := dbtest.NewDatabase()
			Expect(MigrateDatabase(ctx, url)).To(Succeed())

			taken, err := net.Listen("tcp", addressWithAnyPort)
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(taken.Close)

			internalCertFile, internalKeyFile := writeKeyPair(GinkgoTB())
			cfg := authorityConfig(url, internalCertFile, internalKeyFile)
			setListenAddress(&cfg, taken.Addr().String())

			err = run(ctx, cfg, shutdownTimeout, ignoreAddresses)

			Expect(err).To(MatchError(ContainSubstring("listening for " + serverName + ":")))
		},
		Entry("the external API", SpecTimeout(30*time.Second), "external-api", func(cfg *config.Config, address string) {
			cfg.ExternalAPI.ListenAddress = address
		}),
		Entry("the internal API", SpecTimeout(30*time.Second), "internal-api", func(cfg *config.Config, address string) {
			cfg.InternalAPI.ListenAddress = address
		}),
		Entry("the metrics", SpecTimeout(30*time.Second), "metrics", func(cfg *config.Config, address string) {
			cfg.Metrics.ListenAddress = address
		}),
	)
})

var _ = Describe("openDatabase", func() {
	When("the authority migrates on start", func() {
		It("creates the table of the Postgres secret engine", func(ctx SpecContext) {
			url := dbtest.NewDatabase()

			pool, err := openDatabase(ctx, config.Database{URL: url, MigrateOnStart: true})

			Expect(err).NotTo(HaveOccurred())
			pool.Close()
			Expect(secretsTableExists(ctx, url)).To(BeTrue())
		})
	})
})

var _ = Describe("MigrateDatabase", func() {
	It("applies the authority's migrations", func(ctx SpecContext) {
		url := dbtest.NewDatabase()

		Expect(MigrateDatabase(ctx, url)).To(Succeed())

		pool, err := db.OpenMigrated(ctx, url)
		Expect(err).NotTo(HaveOccurred())
		pool.Close()
	})

	It("creates the table of the Postgres secret engine", func(ctx SpecContext) {
		url := dbtest.NewDatabase()

		Expect(MigrateDatabase(ctx, url)).To(Succeed())

		Expect(secretsTableExists(ctx, url)).To(BeTrue())
	})
})

func secretsTableExists(ctx context.Context, url string) bool {
	GinkgoHelper()

	var exists bool

	err := dbtest.Connect(ctx, url).QueryRow(ctx, "SELECT to_regclass('secret_engine.secrets') IS NOT NULL").Scan(&exists)
	Expect(err).NotTo(HaveOccurred())

	return exists
}

// runningAuthority is an authority that serves in the background.
type runningAuthority struct {
	externalAddress  string
	internalAddress  string
	metricsAddress   string
	internalCertFile string
	stop             context.CancelFunc
	runErrorChan     chan error
}

// serveAuthority runs the authority on the database at url and returns once
// both of its APIs and its metrics listen. The external API serves plain
// HTTP. It waits only 100 milliseconds for running requests when it stops,
// so the spec doesn't wait the full shutdown timeout.
func serveAuthority(ctx context.Context, url string) runningAuthority {
	GinkgoHelper()

	runCtx, stop := context.WithCancel(context.Background())
	DeferCleanup(stop)

	internalCertFile, internalKeyFile := writeKeyPair(GinkgoTB())

	addressesChan := make(chan map[string]string, 1)
	runErrorChan := make(chan error, 1)

	go func() {
		cfg := authorityConfig(url, internalCertFile, internalKeyFile)
		runErrorChan <- run(runCtx, cfg, 100*time.Millisecond, sendTo(addressesChan))
	}()

	var addresses map[string]string

	select {
	case addresses = <-addressesChan:
	case err := <-runErrorChan:
		Fail(fmt.Sprintf("the authority stopped before it listened: %v", err))
	case <-ctx.Done():
		Fail("the authority never listened")
	}

	return runningAuthority{
		externalAddress:  addresses["external-api"],
		internalAddress:  addresses["internal-api"],
		metricsAddress:   addresses["metrics"],
		internalCertFile: internalCertFile,
		stop:             stop,
		runErrorChan:     runErrorChan,
	}
}

func authorityConfig(url, internalCertFile, internalKeyFile string) config.Config {
	return config.Config{
		ExternalAPI: config.ExternalAPI{ListenAddress: addressWithAnyPort, TLS: config.ExternalTLS{Disabled: true}},
		InternalAPI: config.InternalAPI{
			ListenAddress: addressWithAnyPort,
			TLS:           config.InternalTLS{CertFile: internalCertFile, KeyFile: internalKeyFile},
		},
		Metrics:  config.Metrics{ListenAddress: addressWithAnyPort},
		Database: config.Database{URL: url},
	}
}

// lockProjects locks the projects table in a transaction that stays open
// until the connection closes.
func lockProjects(ctx context.Context, conn *pgx.Conn) {
	GinkgoHelper()

	_, err := conn.Exec(ctx, "BEGIN; LOCK TABLE projects IN ACCESS EXCLUSIVE MODE")
	Expect(err).NotTo(HaveOccurred())
}

func countSessionsWaitingForALock(ctx context.Context, conn *pgx.Conn) (int, error) {
	var sessions int

	err := conn.QueryRow(ctx,
		"SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock'",
	).Scan(&sessions)

	return sessions, err
}

func clientTrusting(certFile string) *http.Client {
	GinkgoHelper()

	certificate, err := os.ReadFile(certFile) //nolint:gosec // The test wrote the file.
	Expect(err).NotTo(HaveOccurred())

	pool := x509.NewCertPool()
	Expect(pool.AppendCertsFromPEM(certificate)).To(BeTrue())

	transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS13}}
	DeferCleanup(transport.CloseIdleConnections)

	return &http.Client{Transport: transport}
}

// createProject sends a write that only ends with the authority. A request
// whose client gives up would cancel its query without the shutdown.
func createProject(address string, token secret.String) {
	client := authorityv1connect.NewProjectServiceClient(http.DefaultClient, "http://"+address)
	request := connect.NewRequest(&authorityv1.CreateProjectRequest{Name: "waits"})
	request.Header().Set("Authorization", "Bearer "+token.Reveal())

	_, _ = client.CreateProject(context.Background(), request)
}

func signIn(ctx context.Context, url string) secret.String {
	GinkgoHelper()

	pool, err := db.Open(ctx, url)
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(pool.Close)

	data := store.New(pool)
	sessions := service.NewSessions(data.Sessions(), data, random.NewCryptoRandom(), clock.NewSystemClock(), time.Hour)

	token, err := sessions.Issue(ctx, storetest.AddUser(ctx, data, "alice"))
	Expect(err).NotTo(HaveOccurred())

	return token
}
