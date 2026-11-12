package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/tidegate-dev/tidegate/internal/logging"
	"github.com/tidegate-dev/tidegate/internal/systemd"
)

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
