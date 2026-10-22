// Command control-plane runs the tidegate control plane.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tidegate-dev/tidegate/control-plane/internal/config"
	"github.com/tidegate-dev/tidegate/control-plane/internal/ping"
	"github.com/tidegate-dev/tidegate/gen/tidegate/controlplane/v1/controlplanev1connect"
	"github.com/tidegate-dev/tidegate/internal/logging"
	"github.com/tidegate-dev/tidegate/internal/metrics"
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
		slog.Error("control plane stopped", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	metricsHandler, shutdownMetrics, err := metrics.Setup()
	if err != nil {
		return err
	}

	metricsMux := http.NewServeMux()
	metricsMux.Handle("GET /metrics", metricsHandler)

	err = serve(ctx, map[string]*http.Server{
		"api": apiServer(cfg.Server.ListenAddress),
		"metrics": {
			Addr:              cfg.Metrics.ListenAddress,
			Handler:           metricsMux,
			ReadHeaderTimeout: 5 * time.Second,
		},
	})

	return errors.Join(err, shutdownMetrics(context.Background()))
}

func apiServer(address string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle(controlplanev1connect.NewPingServiceHandler(ping.Service{}))

	// gRPC clients need HTTP/2, and the CLI's Connect client uses HTTP/1.1
	// by default, so the server speaks both without TLS.
	var protocols http.Protocols
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	return &http.Server{
		Addr:              address,
		Handler:           mux,
		Protocols:         &protocols,
		ReadHeaderTimeout: 5 * time.Second,
		// Every method is unary with a small request. A streaming method
		// needs its own limit.
		ReadTimeout: 10 * time.Second,
	}
}

// serve runs servers, keyed by a name for the logs, until ctx is done or
// one of them fails, then shuts them all down.
func serve(ctx context.Context, servers map[string]*http.Server) error {
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
		slog.Info("control plane listening", "server", name, "address", listener.Addr().String())
	}

	served := make(chan error, len(servers))

	for server, listener := range listeners {
		go func() {
			served <- server.Serve(listener)
		}()
	}

	var errs []error

	running := len(servers)

	select {
	case err := <-served:
		errs = append(errs, err)
		running--
	case <-ctx.Done():
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
