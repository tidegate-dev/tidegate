package db

import (
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFallsBackToPlaintext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		url  string
		want bool
	}{
		{url: "postgres://phemera@host/phemera", want: true},
		{url: "host=host user=phemera", want: true},
		{url: "postgres://phemera@host/phemera?sslmode=prefer", want: true},
		{url: "postgres://phemera@host/phemera?sslmode=disable", want: false},
		{url: "postgres://phemera@host/phemera?sslmode=require", want: false},
		{url: "postgres://phemera@host/phemera?sslmode=verify-full", want: false},
		{url: "postgres://phemera@host1,host2/phemera?sslmode=verify-full", want: false},
	}

	for _, test := range tests {
		t.Run(test.url, func(t *testing.T) {
			t.Parallel()

			// arrange
			config, err := pgxpool.ParseConfig(test.url)
			require.NoError(t, err)

			// act
			got := fallsBackToPlaintext(config)

			// assert
			assert.Equal(t, test.want, got)
		})
	}
}

func TestFallsBackToPlaintextFollowsPGSSLMODE(t *testing.T) {
	// arrange
	t.Setenv("PGSSLMODE", "disable")

	config, err := pgxpool.ParseConfig("postgres://phemera@host/phemera")
	require.NoError(t, err)

	// act
	got := fallsBackToPlaintext(config)

	// assert
	assert.False(t, got)
}

func TestOpenRejectsAnInvalidURLWithoutTheURL(t *testing.T) {
	t.Parallel()

	// act
	// pgx can't find the password to mask in this URL, because the @ is missing.
	_, err := Open(t.Context(), "postgres://phemera:hunter2host:x/phemera")

	// assert
	assert.EqualError(t, err, "the database URL isn't a valid Postgres connection URL, or a PG variable like PGSSLMODE is invalid")
}

func TestOpenNamesAMissingTLSFile(t *testing.T) {
	t.Parallel()

	// arrange
	missing := filepath.Join(t.TempDir(), "ca.crt")

	// act
	_, err := Open(t.Context(), "postgres://phemera@host/phemera?sslmode=verify-full&sslrootcert="+missing)

	// assert
	assert.EqualError(t, err, "the database URL can't be used, "+
		"because sslrootcert or PGSSLROOTCERT names a file that can't be read: open "+missing+": no such file or directory")
}
