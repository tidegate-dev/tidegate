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

	"github.com/tidegate-dev/tidegate/authority/internal/config"
	"github.com/tidegate-dev/tidegate/authority/internal/ping"
	"github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1/authorityv1connect"
	"github.com/tidegate-dev/tidegate/internal/logging"
	"github.com/tidegate-dev/tidegate/internal/metrics"
	"github.com/tidegate-dev/tidegate/internal/systemd"
)

const shutdownTimeout = 10 * time.Second

func main() {
	cfg, err := config.Load(os.Environ())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid config:\n%v\n", err)
		os.Exit(1)
	}

	slog.SetDefault(logging.New(os.Stderr, cfg.Log.Level, cfg.Log.Format))
	slog.Debug("loaded config", "listen_address", cfg.Server.ListenAddress)

	err = run(cfg)
	if err != nil {
		slog.Error("authority stopped", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
		"api": apiServer(cfg.Server),
		"metrics": {
			Addr:              cfg.Metrics.ListenAddress,
			Handler:           metricsMux,
			ReadHeaderTimeout: 5 * time.Second,
			// Longer than Prometheus's default scrape interval of one minute,
			// so a scraper can reuse its connection.
			IdleTimeout: 90 * time.Second,
		},
	})

	return errors.Join(err, shutdownMetrics(context.Background()))
}

func apiServer(cfg config.Server) *http.Server {
	mux := http.NewServeMux()
	mux.Handle(authorityv1connect.NewPingServiceHandler(ping.Service{}))

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
// one of them fails, then shuts them all down. It tells notifier when all
// servers listen and when the shutdown begins.
func serve(ctx context.Context, notifier *systemd.Notifier, servers map[string]*http.Server) error {
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	for name, server := range servers {
		err := server.Shutdown(shutdownCtx)
		if err != nil {
			errs = append(errs, fmt.Errorf("shutting down %s: %w", name, err))
		}
	}

	for range running {
		err := <-served
		if !errors.Is(err, http.ErrServerClosed) {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
