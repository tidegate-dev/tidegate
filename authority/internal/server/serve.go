package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/phemera-dev/phemera/internal/logging"
	"github.com/phemera-dev/phemera/internal/systemd"
)

// listeningServer is a server with the open listener that it serves on.
type listeningServer struct {
	*http.Server

	listener net.Listener
}

// listen opens a listener on the address of each of servers, keyed by a name
// for the logs. If one address fails, it closes the listeners that are open.
func listen(ctx context.Context, servers map[string]*http.Server) (map[string]listeningServer, error) {
	var listenConfig net.ListenConfig

	listening := make(map[string]listeningServer, len(servers))

	for name, server := range servers {
		listener, err := listenConfig.Listen(ctx, "tcp", server.Addr)
		if err != nil {
			for _, opened := range listening {
				_ = opened.listener.Close()
			}

			return nil, fmt.Errorf("listening for %s: %w", name, err)
		}

		listening[name] = listeningServer{Server: server, listener: listener}
		slog.Info("authority listening", "server", name, "address", listener.Addr().String(), "tls", server.TLSConfig != nil)
	}

	return listening, nil
}

// serve runs servers, keyed by a name for the logs, until ctx is done or
// one of them fails, then shuts them all down. It sets each server's
// BaseContext, waits up to timeout for running requests, then cancels their
// context. It tells notifier when it serves them and when the shutdown
// begins.
func serve(ctx context.Context, notifier *systemd.Notifier, servers map[string]listeningServer, timeout time.Duration) error {
	requests, cancelRequests := context.WithCancel(logging.NewContext(context.Background(), slog.Default()))
	defer cancelRequests()

	for _, server := range servers {
		server.BaseContext = func(net.Listener) context.Context { return requests }
	}

	serveErrorChan := make(chan error, len(servers))

	for _, server := range servers {
		go func() {
			if server.TLSConfig != nil {
				// ServeTLS uses TLSConfig's certificate when the file names are empty.
				serveErrorChan <- server.ServeTLS(server.listener, "", "")
			} else {
				serveErrorChan <- server.Serve(server.listener)
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
