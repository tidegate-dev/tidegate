// Command authority runs the tidegate authority.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/tidegate-dev/tidegate/authority/internal/command"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// stop restores Go's default for the signals, so a second one ends the process.
	context.AfterFunc(ctx, stop)

	err := command.New().Run(ctx, os.Args)
	if errors.Is(err, command.ErrInvalidConfig) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err != nil {
		slog.Error("authority stopped", "error", err)
		os.Exit(1)
	}
}
