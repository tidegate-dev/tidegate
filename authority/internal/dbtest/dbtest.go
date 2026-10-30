// Package dbtest gives integration tests an empty Postgres database of their
// own.
package dbtest

import (
	"context"
	"crypto/rand"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// URLVariable names the environment variable with the URL of the Postgres
// server that integration tests use. The tests connect to the database it
// names only to create their own, so its user needs the CREATEDB privilege.
const URLVariable = "TIDEGATE_TEST_DATABASE_URL"

// NewDatabase creates an empty database, drops it when the test ends and
// returns its URL. It fails the test if URLVariable isn't set.
func NewDatabase(t testing.TB) string {
	t.Helper()

	serverURL := os.Getenv(URLVariable)
	if serverURL == "" {
		t.Fatal(URLVariable + " isn't set. Run the integration tests with just test-integration")
	}

	databaseURL, err := url.Parse(serverURL)
	if err != nil || (databaseURL.Scheme != "postgres" && databaseURL.Scheme != "postgresql") {
		t.Fatal(URLVariable + " must be a postgres:// URL")
	}

	name := "test_" + strings.ToLower(rand.Text())
	quoted := pgx.Identifier{name}.Sanitize()

	exec(t, serverURL, "CREATE DATABASE "+quoted)
	t.Cleanup(func() {
		exec(t, serverURL, "DROP DATABASE "+quoted+" WITH (FORCE)")
	})

	databaseURL.Path = "/" + name

	return databaseURL.String()
}

func exec(t testing.TB, serverURL, sql string) {
	t.Helper()

	// Cleanup runs after the test's context is canceled.
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, serverURL)
	if err != nil {
		t.Fatalf("connecting to the database in %s: %v", URLVariable, err)
	}

	defer func() {
		_ = conn.Close(ctx)
	}()

	_, err = conn.Exec(ctx, sql)
	if err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}
