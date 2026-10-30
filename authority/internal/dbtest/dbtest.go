//go:build integration

// Package dbtest gives Ginkgo specs an empty Postgres database of their own.
package dbtest

import (
	"context"
	"crypto/rand"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// URLVariable names the environment variable with the URL of the Postgres
// server that integration tests use. The tests connect to the database it
// names only to create their own, so its user needs the CREATEDB privilege.
const URLVariable = "TIDEGATE_TEST_DATABASE_URL"

// NewDatabase creates an empty database, drops it when the spec ends and
// returns its URL. It fails the spec if URLVariable isn't set.
func NewDatabase() string {
	GinkgoHelper()

	serverURL := os.Getenv(URLVariable)
	Expect(serverURL).NotTo(BeEmpty(), URLVariable+" isn't set. Run the integration tests with just test-integration")

	// url.Parse's error contains the URL, password included.
	databaseURL, err := url.Parse(serverURL)
	if err != nil || (databaseURL.Scheme != "postgres" && databaseURL.Scheme != "postgresql") {
		Fail(URLVariable + " must be a postgres:// URL")
	}

	// pgx lets these replace the database in the path.
	if query := databaseURL.Query(); query.Has("dbname") || query.Has("database") {
		Fail(URLVariable + " must not name a database in its query")
	}

	name := "test_" + strings.ToLower(rand.Text())
	quoted := pgx.Identifier{name}.Sanitize()

	exec(serverURL, "CREATE DATABASE "+quoted)
	DeferCleanup(exec, serverURL, "DROP DATABASE "+quoted+" WITH (FORCE)")

	databaseURL.Path = "/" + name

	return databaseURL.String()
}

func exec(serverURL, sql string) {
	GinkgoHelper()

	// Cleanup runs after the spec's context is canceled.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, serverURL)
	Expect(err).NotTo(HaveOccurred(), "connecting to the database in "+URLVariable)

	defer func() {
		_ = conn.Close(ctx)
	}()

	_, err = conn.Exec(ctx, sql)
	Expect(err).NotTo(HaveOccurred(), sql)
}
