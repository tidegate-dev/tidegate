// Command ssh-proxy runs the phemera ssh-proxy.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/phemera-dev/phemera/internal/logging"
	"github.com/phemera-dev/phemera/ssh-proxy/internal/config"
	"github.com/phemera-dev/phemera/ssh-proxy/internal/server"
)

func main() {
	if len(os.Args) > 1 {
		fmt.Fprintln(os.Stderr, "ssh-proxy takes no arguments. Its settings are environment variables that start with PHEMERA_SSH_PROXY__")
		os.Exit(1)
	}

	cfg, err := config.Load(os.Environ())
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid config:\n%v\n", err)
		os.Exit(1)
	}

	slog.SetDefault(logging.New(os.Stderr, cfg.Log.Level, cfg.Log.Format))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// stop restores Go's default for the signals, so a second one ends the process.
	context.AfterFunc(ctx, stop)

	err = server.Run(ctx, cfg)
	if err != nil {
		slog.Error("ssh-proxy stopped", "error", err)
		os.Exit(1)
	}
}
