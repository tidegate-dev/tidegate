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

const unreachableDatabaseURL = "postgres://tidegate@127.0.0.1:1/tidegate?sslmode=disable"

func TestRunStopsAtAMissingExternalCertificateBeforeTheDatabase(t *testing.T) {
	t.Parallel()

	// arrange
	_, keyFile := writeKeyPair(t)
	missing := filepath.Join(t.TempDir(), "tls.crt")
	cfg := config.Config{
		ExternalAPI: config.ExternalAPI{TLS: config.ExternalTLS{CertFile: missing, KeyFile: keyFile}},
		Database:    config.Database{URL: unreachableDatabaseURL},
	}

	// act
	err := run(t.Context(), cfg, time.Second)

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), missing)
	assert.NotContains(t, err.Error(), "database")
}

func TestRunStopsAtAMissingInternalCertificateBeforeTheDatabase(t *testing.T) {
	t.Parallel()

	// arrange
	_, keyFile := writeKeyPair(t)
	missing := filepath.Join(t.TempDir(), "tls.crt")
	cfg := config.Config{
		ExternalAPI: config.ExternalAPI{TLS: config.ExternalTLS{Disabled: true}},
		InternalAPI: config.InternalAPI{TLS: config.InternalTLS{CertFile: missing, KeyFile: keyFile}},
		Database:    config.Database{URL: unreachableDatabaseURL},
	}

	// act
	err := run(t.Context(), cfg, time.Second)

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), missing)
	assert.NotContains(t, err.Error(), "database")
}

func TestRunWarnsWhenTLSIsDisabledForTheExternalAPI(t *testing.T) {
	// The warning goes to the default logger, which is global.

	// arrange
	var logs bytes.Buffer

	previous := slog.Default()

	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})

	certFile, keyFile := writeKeyPair(t)
	cfg := config.Config{
		ExternalAPI: config.ExternalAPI{TLS: config.ExternalTLS{Disabled: true}},
		InternalAPI: config.InternalAPI{TLS: config.InternalTLS{CertFile: certFile, KeyFile: keyFile}},
		Database:    config.Database{URL: unreachableDatabaseURL},
	}

	// act
	_ = run(t.Context(), cfg, time.Second)

	// assert
	assert.Contains(t, logs.String(), "level=WARN")
	assert.Contains(t, logs.String(), "the external API serves plain HTTP")
}
