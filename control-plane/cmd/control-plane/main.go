// Command control-plane runs the tidegate control plane.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tidegate-dev/tidegate/control-plane/internal/config"
	"github.com/tidegate-dev/tidegate/control-plane/internal/logging"
	"github.com/tidegate-dev/tidegate/control-plane/internal/ping"
	"github.com/tidegate-dev/tidegate/gen/tidegate/controlplane/v1/controlplanev1connect"
)

const shutdownTimeout = 10 * time.Second

func main() {
	cfg, err := config.Load(os.Environ())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid config:\n%v\n", err)
		os.Exit(1)
	}

	slog.SetDefault(logging.New(os.Stderr, cfg.Log))
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

	mux := http.NewServeMux()
	mux.Handle(controlplanev1connect.NewPingServiceHandler(ping.Service{}))

	// gRPC clients need HTTP/2, and the CLI's Connect client uses HTTP/1.1
	// by default, so the server speaks both without TLS.
	var protocols http.Protocols
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	server := &http.Server{
		Addr:              cfg.Server.ListenAddress,
		Handler:           mux,
		Protocols:         &protocols,
		ReadHeaderTimeout: 5 * time.Second,
	}

	served := make(chan error, 1)

	go func() {
		served <- server.ListenAndServe()
	}()

	slog.Info("control plane listening", "address", cfg.Server.ListenAddress)

	select {
	case err := <-served:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	err := server.Shutdown(shutdownCtx)
	if err != nil {
		return fmt.Errorf("shutting down: %w", err)
	}

	err = <-served
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
