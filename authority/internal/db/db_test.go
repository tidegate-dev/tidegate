package db

import (
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
		{url: "postgres://tidegate@host/tidegate", want: true},
		{url: "host=host user=tidegate", want: true},
		{url: "postgres://tidegate@host/tidegate?sslmode=prefer", want: true},
		{url: "postgres://tidegate@host/tidegate?sslmode=disable", want: false},
		{url: "postgres://tidegate@host/tidegate?sslmode=require", want: false},
		{url: "postgres://tidegate@host/tidegate?sslmode=verify-full", want: false},
		{url: "postgres://tidegate@host1,host2/tidegate?sslmode=verify-full", want: false},
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

	config, err := pgxpool.ParseConfig("postgres://tidegate@host/tidegate")
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
	_, err := Open(t.Context(), "postgres://tidegate:hunter2host:x/tidegate")

	// assert
	assert.EqualError(t, err, "the database URL isn't a valid Postgres connection URL")
}
