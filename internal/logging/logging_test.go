package logging_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/internal/logging"
)

func TestNewWritesJSON(t *testing.T) {
	t.Parallel()

	// arrange
	var out bytes.Buffer

	logger := logging.New(&out, slog.LevelInfo, logging.FormatJSON)

	// act
	logger.Info("started", "address", "127.0.0.1:8080")

	// assert
	var line map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &line))
	assert.Equal(t, "started", line["msg"])
	assert.Equal(t, "127.0.0.1:8080", line["address"])
}

func TestNewWritesText(t *testing.T) {
	t.Parallel()

	// arrange
	var out bytes.Buffer

	logger := logging.New(&out, slog.LevelInfo, logging.FormatText)

	// act
	logger.Info("started", "address", "127.0.0.1:8080")

	// assert
	assert.Contains(t, out.String(), `msg=started address=127.0.0.1:8080`)
}

func TestNewDropsLinesBelowTheLevel(t *testing.T) {
	t.Parallel()

	// arrange
	var out bytes.Buffer

	logger := logging.New(&out, slog.LevelWarn, logging.FormatJSON)

	// act
	logger.Info("started")

	// assert
	assert.Empty(t, out.String())
}

func TestFromContextReturnsTheLoggerFromNewContext(t *testing.T) {
	t.Parallel()

	// arrange
	logger := logging.New(&bytes.Buffer{}, slog.LevelInfo, logging.FormatJSON)
	ctx := logging.NewContext(t.Context(), logger)

	// act
	got := logging.FromContext(ctx)

	// assert
	assert.Same(t, logger, got)
}

func TestFromContextFallsBackToTheDefault(t *testing.T) {
	t.Parallel()

	// act
	got := logging.FromContext(t.Context())

	// assert
	assert.Same(t, slog.Default(), got)
}
