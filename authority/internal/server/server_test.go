package server

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phemera-dev/phemera/authority/internal/config"
	"github.com/phemera-dev/phemera/internal/certreload"
)

const unreachableDatabaseURL = "postgres://phemera@127.0.0.1:1/phemera?sslmode=disable"

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

	// run would wait for the unreachable database.
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	// act
	_ = run(ctx, cfg, time.Second)

	// assert
	assert.Contains(t, logs.String(), "level=WARN")
	assert.Contains(t, logs.String(), "the external API serves plain HTTP")
}

func TestRunIsNotReadyWhileItWaitsForTheDatabase(t *testing.T) {
	// listenAsSystemd sets NOTIFY_SOCKET, which is global.

	// arrange
	socket := listenAsSystemd(t)
	certFile, keyFile := writeKeyPair(t)
	cfg := config.Config{
		ExternalAPI: config.ExternalAPI{TLS: config.ExternalTLS{Disabled: true}},
		InternalAPI: config.InternalAPI{TLS: config.InternalTLS{CertFile: certFile, KeyFile: keyFile}},
		Database:    config.Database{URL: unreachableDatabaseURL},
	}

	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()

	// act
	err := run(ctx, cfg, time.Second)

	// assert
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Contains(t, err.Error(), "stopped waiting for the database")
	require.NoError(t, socket.SetReadDeadline(time.Now().Add(100*time.Millisecond)))
	_, _, err = socket.ReadFrom(make([]byte, 64))
	assert.ErrorIs(t, err, os.ErrDeadlineExceeded, "run told systemd something")
}

func TestNewAuthorityServersGivesEachServerTheListenAddressFromItsPartOfTheConfig(t *testing.T) {
	t.Parallel()

	// arrange
	// A pool connects to the database only when a query needs it.
	pool, err := pgxpool.New(t.Context(), unreachableDatabaseURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	certFile, keyFile := writeKeyPair(t)
	internalCertificate, err := certreload.Load(certFile, keyFile)
	require.NoError(t, err)

	cfg := config.Config{
		ExternalAPI: config.ExternalAPI{ListenAddress: "external.example:1"},
		InternalAPI: config.InternalAPI{ListenAddress: "internal.example:2"},
		Metrics:     config.Metrics{ListenAddress: "metrics.example:3"},
	}

	// act
	servers, err := newAuthorityServers(cfg, pool, nil, internalCertificate, http.NotFoundHandler())

	// assert
	require.NoError(t, err)
	assert.Equal(t, "external.example:1", servers.externalAPI.Addr)
	assert.Equal(t, "internal.example:2", servers.internalAPI.Addr)
	assert.Equal(t, "metrics.example:3", servers.metrics.Addr)
}
