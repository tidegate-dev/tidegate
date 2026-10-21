// Package logging builds tidegate's loggers and passes them to request
// handlers through a context.
package logging

import (
	"context"
	"io"
	"log/slog"
)

// Format is how log lines are written.
type Format string

const (
	// FormatJSON writes one JSON object per line, for log collectors.
	FormatJSON Format = "json"
	// FormatText writes key=value pairs, which are easier to read locally.
	FormatText Format = "text"
)

// New returns a logger that writes to w in format and drops lines below
// level.
func New(w io.Writer, level slog.Level, format Format) *slog.Logger {
	options := &slog.HandlerOptions{Level: level}

	if format == FormatText {
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
