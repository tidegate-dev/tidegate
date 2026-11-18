// Package server runs the authority's API servers and metrics server.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/The127/go-clock"
	"github.com/The127/go-random"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tidegate-dev/tidegate/authority/internal/config"
	"github.com/tidegate-dev/tidegate/authority/internal/db"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
	"github.com/tidegate-dev/tidegate/authority/internal/store"
	"github.com/tidegate-dev/tidegate/internal/metrics"
	"github.com/tidegate-dev/tidegate/internal/systemd"
)

const shutdownTimeout = 10 * time.Second

// Run serves the authority's APIs and metrics until ctx ends or a server
// fails.
func Run(ctx context.Context, cfg config.Config) error {
	return run(ctx, cfg, shutdownTimeout)
}

func run(ctx context.Context, cfg config.Config, timeout time.Duration) error {
	externalCertificate, err := loadExternalCertificate(cfg.ExternalAPI.TLS)
	if err != nil {
		return err
	}

	if externalCertificate == nil {
		slog.Warn("the external API serves plain HTTP, because " + config.ExternalTLSDisabledVariable() + " is true. " +
			"Only use this behind a proxy that terminates TLS")
	}

	internalCertificate, err := loadInternalCertificate(cfg.InternalAPI.TLS)
	if err != nil {
		return err
	}

	pool, err := openDatabase(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()

	rnd := random.NewCryptoRandom()
	clk := clock.NewSystemClock()
	data := store.New(pool)

	authenticator, err := service.NewAuthenticator(data.Users(), rnd)
	if err != nil {
		return err
	}

	sessions := service.NewSessions(data.Sessions(), data, rnd, clk, cfg.Login.TokenLifetime)
	logins := service.NewLogins(authenticator, sessions)
	dummies := service.NewDummies(data.Dummies(), data)

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

	stopWatchingExternal, err := watchCertificate(ctx, externalCertificate)
	if err != nil {
		return errors.Join(err, shutdownMetrics(context.Background()))
	}
	defer stopWatchingExternal()

	stopWatchingInternal, err := watchCertificate(ctx, internalCertificate)
	if err != nil {
		return errors.Join(err, shutdownMetrics(context.Background()))
	}
	defer stopWatchingInternal()

	err = serve(ctx, notifier, map[string]*http.Server{
		"external-api": externalAPIServer(cfg.ExternalAPI.ListenAddress, externalCertificate, clk, dummies, logins, sessions),
		"internal-api": internalAPIServer(cfg.InternalAPI.ListenAddress, internalCertificate),
		"metrics":      metricsServer(cfg.Metrics.ListenAddress, metricsHandler),
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
