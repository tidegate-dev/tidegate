// Package logging builds the control plane's logger from the config and
// passes it to request handlers through a context.
package logging

import (
	"context"
	"io"
	"log/slog"

	"github.com/tidegate-dev/tidegate/control-plane/internal/config"
)

// New returns a logger that writes to w in the format and from the level
// that cfg sets.
func New(w io.Writer, cfg config.Log) *slog.Logger {
	options := &slog.HandlerOptions{Level: cfg.Level}

	if cfg.Format == config.LogFormatText {
		return slog.New(slog.NewTextHandler(w, options))
	}

	return slog.New(slog.NewJSONHandler(w, options))
}

type contextKey struct{}

// NewContext returns a copy of ctx that carries logger.
func NewContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromContext returns the logger that ctx carries, or slog.Default if it
// carries none.
func FromContext(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(contextKey{}).(*slog.Logger)
	if !ok {
		return slog.Default()
	}

	return logger
}
