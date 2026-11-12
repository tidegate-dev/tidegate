package command

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/validate"
	"github.com/The127/go-clock"
	"github.com/The127/go-random"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tidegate-dev/tidegate/authority/internal/api"
	"github.com/tidegate-dev/tidegate/authority/internal/config"
	"github.com/tidegate-dev/tidegate/authority/internal/db"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
	"github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1/authorityv1connect"
	"github.com/tidegate-dev/tidegate/internal/certreload"
	"github.com/tidegate-dev/tidegate/internal/handlererrors"
	"github.com/tidegate-dev/tidegate/internal/logging"
	"github.com/tidegate-dev/tidegate/internal/metrics"
	"github.com/tidegate-dev/tidegate/internal/requestlog"
	"github.com/tidegate-dev/tidegate/internal/systemd"
)

func run(ctx context.Context, cfg config.Config, timeout time.Duration) error {
	certificate, err := loadCertificate(cfg.Server.TLS)
	if err != nil {
		return err
	}

	if certificate == nil {
		slog.Warn("the API serves plain HTTP, because TIDEGATE_AUTHORITY__SERVER__TLS__DISABLED is true. " +
			"Only use this behind a proxy that terminates TLS")
	}

	if cfg.Database.MigrateOnStart {
		err := db.MigrateAt(ctx, cfg.Database.URL)
		if err != nil {
			return err
		}
	}

	pool, err := db.OpenMigrated(ctx, cfg.Database.URL)
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

	if certificate != nil {
		registration, err := certificate.ReportExpiry()
		if err != nil {
			return errors.Join(err, shutdownMetrics(context.Background()))
		}

		defer func() {
			_ = registration.Unregister()
		}()

		go certificate.Watch(ctx)
	}

	metricsMux := http.NewServeMux()
	metricsMux.Handle("GET /metrics", metricsHandler)

	err = serve(ctx, notifier, map[string]*http.Server{
		"api": apiServer(cfg.Server.ListenAddress, certificate, pool, logins, tokens),
		"metrics": {
			Addr:              cfg.Metrics.ListenAddress,
			Handler:           metricsMux,
			ReadHeaderTimeout: 5 * time.Second,
			// Longer than Prometheus's default scrape interval of one minute,
			// so a scraper can reuse its connection.
			IdleTimeout: 90 * time.Second,
		},
	}, timeout)

	return errors.Join(err, shutdownMetrics(context.Background()))
}

// requestlog comes first, so it logs the code the client gets after
// handlererrors has changed it.
func interceptors(clk clock.Service, sessions api.SessionLookup) []connect.Interceptor {
	return []connect.Interceptor{
		requestlog.Interceptor(clk),
		handlererrors.Interceptor(),
		api.NewSessionInterceptor(sessions,
			authorityv1connect.LoginServiceLoginProcedure,
			authorityv1connect.PingServicePingProcedure,
		),
		validate.NewInterceptor(),
	}
}

// apiServer serves plain HTTP if certificate is nil.
func apiServer(
	address string,
	certificate *certreload.Certificate,
	pool *pgxpool.Pool,
	logins *service.Logins,
	sessions api.SessionLookup,
) *http.Server {
	options := connect.WithInterceptors(interceptors(clock.NewSystemClock(), sessions)...)

	mux := http.NewServeMux()
	mux.Handle(authorityv1connect.NewPingServiceHandler(api.PingService{}, options))
	mux.Handle(authorityv1connect.NewDummyServiceHandler(api.NewDummyService(service.NewDummies(pool)), options))
	mux.Handle(api.NewLoginServiceHandler(logins, options))

	// gRPC clients need HTTP/2, and the CLI's Connect client uses HTTP/1.1
	// by default.
	var protocols http.Protocols
	protocols.SetHTTP1(true)

	var tlsConfig *tls.Config

	if certificate != nil {
		tlsConfig = &tls.Config{GetCertificate: certificate.Get}

		protocols.SetHTTP2(true)
	} else {
		protocols.SetUnencryptedHTTP2(true)
	}

	return &http.Server{
		Addr:              address,
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

	serveErrorChan := make(chan error, len(servers))

	for server, listener := range listeners {
		go func() {
			if server.TLSConfig != nil {
				// ServeTLS uses TLSConfig's certificate when the file names are empty.
				serveErrorChan <- server.ServeTLS(listener, "", "")
			} else {
				serveErrorChan <- server.Serve(listener)
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
	case err := <-serveErrorChan:
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
		err := <-serveErrorChan
		if !errors.Is(err, http.ErrServerClosed) {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
