// Command authority runs the tidegate authority.
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/urfave/cli/v3"

	"github.com/tidegate-dev/tidegate/authority/internal/config"
	"github.com/tidegate-dev/tidegate/authority/internal/db"
	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
	"github.com/tidegate-dev/tidegate/authority/internal/dummy"
	"github.com/tidegate-dev/tidegate/authority/internal/ping"
	"github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1/authorityv1connect"
	"github.com/tidegate-dev/tidegate/internal/handlererrors"
	"github.com/tidegate-dev/tidegate/internal/logging"
	"github.com/tidegate-dev/tidegate/internal/metrics"
	"github.com/tidegate-dev/tidegate/internal/systemd"
)

const shutdownTimeout = 10 * time.Second

// errInvalidConfig marks errors that main prints as they are, because the
// logger is only set up from a valid config.
var errInvalidConfig = errors.New("invalid config")

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// stop restores Go's default for the signals, so a second one ends the process.
	context.AfterFunc(ctx, stop)

	err := newCommand().Run(ctx, os.Args)
	if errors.Is(err, errInvalidConfig) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err != nil {
		slog.Error("authority stopped", "error", err)
		os.Exit(1)
	}
}

func newCommand() *cli.Command {
	return &cli.Command{
		Name:  "authority",
		Usage: "run the tidegate authority",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg, err := setUp(cmd)
			if err != nil {
				return err
			}

			return run(ctx, cfg)
		},
		Commands: []*cli.Command{
			{
				Name:  "migrate",
				Usage: "apply the database migrations, then exit",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					cfg, err := setUp(cmd)
					if err != nil {
						return err
					}

					return migrate(ctx, cfg.Database)
				},
			},
		},
	}
}

// setUp rejects arguments, loads the config and sets up logging from it.
// The commands call it themselves, so --help works without a config.
func setUp(cmd *cli.Command) (config.Config, error) {
	if cmd.Args().Present() {
		return config.Config{}, fmt.Errorf("unknown command %q", cmd.Args().First())
	}

	cfg, err := config.Load(os.Environ())
	if err != nil {
		return config.Config{}, fmt.Errorf("%w:\n%w", errInvalidConfig, err)
	}

	slog.SetDefault(logging.New(os.Stderr, cfg.Log.Level, cfg.Log.Format))
	slog.Debug("loaded config", "listen_address", cfg.Server.ListenAddress)

	return cfg, nil
}

func migrate(ctx context.Context, cfg config.Database) error {
	pool, err := db.Open(ctx, cfg.URL)
	if err != nil {
		return err
	}
	defer pool.Close()

	return db.Migrate(ctx, pool)
}

func run(ctx context.Context, cfg config.Config) error {
	pool, err := openDatabase(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()

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

	metricsMux := http.NewServeMux()
	metricsMux.Handle("GET /metrics", metricsHandler)

	err = serve(ctx, notifier, map[string]*http.Server{
		"api": apiServer(cfg.Server, pool),
		"metrics": {
			Addr:              cfg.Metrics.ListenAddress,
			Handler:           metricsMux,
			ReadHeaderTimeout: 5 * time.Second,
			// Longer than Prometheus's default scrape interval of one minute,
			// so a scraper can reuse its connection.
			IdleTimeout: 90 * time.Second,
		},
	}, shutdownTimeout)

	return errors.Join(err, shutdownMetrics(context.Background()))
}

func openDatabase(ctx context.Context, cfg config.Database) (*pgxpool.Pool, error) {
	pool, err := db.Open(ctx, cfg.URL)
	if err != nil {
		return nil, err
	}

	if cfg.MigrateOnStart {
		err = db.Migrate(ctx, pool)
	} else {
		err = db.CheckSchema(ctx, pool)
		if errors.Is(err, db.ErrSchemaTooOld) {
			err = fmt.Errorf("%w. Run authority migrate first", err)
		}
	}

	if err != nil {
		pool.Close()

		return nil, err
	}

	return pool, nil
}

func apiServer(cfg config.Server, pool *pgxpool.Pool) *http.Server {
	options := connect.WithInterceptors(handlererrors.Interceptor())

	mux := http.NewServeMux()
	mux.Handle(authorityv1connect.NewPingServiceHandler(ping.Service{}, options))
	mux.Handle(authorityv1connect.NewDummyServiceHandler(dummy.NewService(sqlc.New(pool)), options))

	// gRPC clients need HTTP/2, and the CLI's Connect client uses HTTP/1.1
	// by default.
	var protocols http.Protocols
	protocols.SetHTTP1(true)

	var tlsConfig *tls.Config

	if cfg.Certificate != nil {
		tlsConfig = &tls.Config{Certificates: []tls.Certificate{*cfg.Certificate}}

		protocols.SetHTTP2(true)
	} else {
		protocols.SetUnencryptedHTTP2(true)
	}

	return &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           mux,
		Protocols:         &protocols,
		TLSConfig:         tlsConfig,
		ReadHeaderTimeout: 5 * time.Second,
		// Every method is unary with a small request. A streaming method
		// needs its own limit.
		ReadTimeout: 10 * time.Second,
	}
}

// serve runs servers, keyed by a name for the logs, until ctx is done or
// one of them fails, then shuts them all down. It sets each server's
// BaseContext, waits up to timeout for running requests, then cancels their
// context. It tells notifier when all servers listen and when the shutdown
// begins.
func serve(ctx context.Context, notifier *systemd.Notifier, servers map[string]*http.Server, timeout time.Duration) error {
	requests, cancelRequests := context.WithCancel(logging.NewContext(context.Background(), slog.Default()))
	defer cancelRequests()

	for _, server := range servers {
		server.BaseContext = func(net.Listener) context.Context { return requests }
	}

	var listenConfig net.ListenConfig

	listeners := make(map[*http.Server]net.Listener, len(servers))

	for name, server := range servers {
		listener, err := listenConfig.Listen(ctx, "tcp", server.Addr)
		if err != nil {
			for _, l := range listeners {
				_ = l.Close()
			}

			return fmt.Errorf("listening for %s: %w", name, err)
		}

		listeners[server] = listener
		slog.Info("authority listening", "server", name, "address", listener.Addr().String(), "tls", server.TLSConfig != nil)
	}

	served := make(chan error, len(servers))

	for server, listener := range listeners {
		go func() {
			if server.TLSConfig != nil {
				// ServeTLS uses TLSConfig's certificate when the file names are empty.
				served <- server.ServeTLS(listener, "", "")
			} else {
				served <- server.Serve(listener)
			}
		}()
	}

	err := notifier.Ready()
	if err != nil {
		slog.Error("telling systemd that the authority is ready", "error", err)
	}

	var errs []error

	running := len(servers)

	select {
	case err := <-served:
		errs = append(errs, err)
		running--
	case <-ctx.Done():
	}

	err = notifier.Stopping()
	if err != nil {
		slog.Error("telling systemd that the authority is stopping", "error", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for name, server := range servers {
		err := server.Shutdown(shutdownCtx)
		if err != nil {
			errs = append(errs, fmt.Errorf("shutting down %s: %w", name, err))
		}
	}

	// Shutdown gives up at the timeout and leaves the handlers running.
	cancelRequests()

	for range running {
		err := <-served
		if !errors.Is(err, http.ErrServerClosed) {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
