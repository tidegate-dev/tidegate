// Package server runs the authority's API server and metrics server.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/The127/go-random"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tidegate-dev/tidegate/authority/internal/config"
	"github.com/tidegate-dev/tidegate/authority/internal/db"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
	"github.com/tidegate-dev/tidegate/internal/metrics"
	"github.com/tidegate-dev/tidegate/internal/systemd"
)

const shutdownTimeout = 10 * time.Second

// Run serves the authority's API and metrics until ctx ends or a server
// fails.
func Run(ctx context.Context, cfg config.Config) error {
	return run(ctx, cfg, shutdownTimeout)
}

func run(ctx context.Context, cfg config.Config, timeout time.Duration) error {
	certificate, err := loadCertificate(cfg.Server.TLS)
	if err != nil {
		return err
	}

	if certificate == nil {
		slog.Warn("the API serves plain HTTP, because TIDEGATE_AUTHORITY__SERVER__TLS__DISABLED is true. " +
			"Only use this behind a proxy that terminates TLS")
	}

	pool, err := openDatabase(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()

	rnd := random.NewCryptoRandom()

	authenticator, err := service.NewAuthenticator(pool, rnd)
	if err != nil {
		return err
	}

	tokens := service.NewSessions(pool, rnd, cfg.Login.TokenLifetime)
	logins := service.NewLogins(authenticator, tokens)

	notifier, err := systemd.Open()
	if err != nil {
		return err
	}

	defer func() {
		_ = notifier.Close()
	}()

	metricsHandler, shutdownMetrics, err := metrics.Setup()
	if err != nil {
		return err
	}

	stopWatching, err := watchCertificate(ctx, certificate)
	if err != nil {
		return errors.Join(err, shutdownMetrics(context.Background()))
	}
	defer stopWatching()

	err = serve(ctx, notifier, map[string]*http.Server{
		"api":     apiServer(cfg.Server.ListenAddress, certificate, pool, logins, tokens),
		"metrics": metricsServer(cfg.Metrics.ListenAddress, metricsHandler),
	}, timeout)

	return errors.Join(err, shutdownMetrics(context.Background()))
}

func openDatabase(ctx context.Context, cfg config.Database) (*pgxpool.Pool, error) {
	if cfg.MigrateOnStart {
		err := db.MigrateAt(ctx, cfg.URL)
		if err != nil {
			return nil, err
		}
	}

	return db.OpenMigrated(ctx, cfg.URL)
}
