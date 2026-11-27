// Package server runs the authority's API servers and the server for its
// metrics and health checks.
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

	"github.com/phemera-dev/phemera/authority/internal/config"
	"github.com/phemera-dev/phemera/authority/internal/db"
	"github.com/phemera-dev/phemera/authority/internal/secretengine/postgres"
	"github.com/phemera-dev/phemera/authority/internal/service"
	"github.com/phemera-dev/phemera/authority/internal/store"
	"github.com/phemera-dev/phemera/internal/metrics"
	"github.com/phemera-dev/phemera/internal/secretengine"
	"github.com/phemera-dev/phemera/internal/systemd"
)

const shutdownTimeout = 10 * time.Second

// Assets store this name, so changing it needs a migration.
const builtInEngineName = "phemera"

// Run serves the authority's APIs, metrics and health checks until ctx ends
// or a server fails.
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
	projects := service.NewProjects(data.Projects(), data)
	roles := service.NewRoles(data.Projects(), data)
	engines := secretengine.NewEngines(map[string]secretengine.Engine{builtInEngineName: postgres.New(pool)})
	assets := service.NewAssets(data.Projects(), data.Assets(), data, engines, builtInEngineName)
	bindings := service.NewBindings(data.Projects(), data.Roles(), data.Users(), data.Bindings(), data)

	tickets := service.NewTickets(data)

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

	watching, stopWatching := context.WithCancel(ctx)
	defer stopWatching()

	stopReportingExternal, err := watchCertificate(watching, externalCertificate)
	if err != nil {
		return errors.Join(err, shutdownMetrics(context.Background()))
	}
	defer stopReportingExternal()

	stopReportingInternal, err := watchCertificate(watching, internalCertificate)
	if err != nil {
		return errors.Join(err, shutdownMetrics(context.Background()))
	}
	defer stopReportingInternal()

	err = serve(ctx, notifier, map[string]*http.Server{
		"external-api": externalAPIServer(cfg.ExternalAPI.ListenAddress, externalCertificate, clk, logins, projects, roles, assets, bindings, tickets, sessions),
		"internal-api": internalAPIServer(cfg.InternalAPI.ListenAddress, internalCertificate),
		"metrics":      metricsServer(cfg.Metrics.ListenAddress, metricsHandler, pool),
	}, timeout)

	return errors.Join(err, shutdownMetrics(context.Background()))
}

// openDatabase waits until the database is available, migrates it if cfg
// says so and checks its schema.
func openDatabase(ctx context.Context, cfg config.Database) (*pgxpool.Pool, error) {
	pool, err := db.OpenWhenAvailable(ctx, cfg.URL)
	if err != nil {
		return nil, err
	}

	err = prepareSchema(ctx, pool, cfg.MigrateOnStart)
	if err != nil {
		pool.Close()

		return nil, err
	}

	return pool, nil
}

func prepareSchema(ctx context.Context, pool *pgxpool.Pool, migrate bool) error {
	if migrate {
		err := migrateDatabase(ctx, pool)
		if err != nil {
			return err
		}
	}

	return db.CheckSchema(ctx, pool)
}

// MigrateDatabase applies the authority's migrations to the database at url.
// Then it creates the schema of the Postgres secret engine, if it's missing.
func MigrateDatabase(ctx context.Context, url string) error {
	pool, err := db.Open(ctx, url)
	if err != nil {
		return err
	}
	defer pool.Close()

	return migrateDatabase(ctx, pool)
}

func migrateDatabase(ctx context.Context, pool *pgxpool.Pool) error {
	err := db.Migrate(ctx, pool)
	if err != nil {
		return err
	}

	return postgres.CreateSchema(ctx, pool)
}
