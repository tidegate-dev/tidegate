package server

import (
	"bytes"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/authority/internal/config"
)

func TestRunStopsAtAMissingCertificateBeforeTheDatabase(t *testing.T) {
	t.Parallel()

	// arrange
	_, keyFile := writeKeyPair(t)
	missing := filepath.Join(t.TempDir(), "tls.crt")
	cfg := config.Config{
		Server:   config.Server{TLS: config.TLS{CertFile: missing, KeyFile: keyFile}},
		Database: config.Database{URL: "postgres://tidegate@127.0.0.1:1/tidegate?sslmode=disable"},
	}

	// act
	err := run(t.Context(), cfg, time.Second)

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), missing)
	assert.NotContains(t, err.Error(), "database")
}

func TestRunWarnsWhenTLSIsDisabled(t *testing.T) {
	// The warning goes to the default logger, which is global.

	// arrange
	var logs bytes.Buffer

	previous := slog.Default()

	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})

	cfg := config.Config{
		Server:   config.Server{TLS: config.TLS{Disabled: true}},
		Database: config.Database{URL: "postgres://tidegate@127.0.0.1:1/tidegate?sslmode=disable"},
	}

	// act
	_ = run(t.Context(), cfg, time.Second)

	// assert
	assert.Contains(t, logs.String(), "level=WARN")
	assert.Contains(t, logs.String(), "plain HTTP")
}
