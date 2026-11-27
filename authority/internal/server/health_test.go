package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phemera-dev/phemera/internal/logging"
)

var errDatabaseDown = errors.New("dial tcp 10.0.0.5:5432: connect: connection refused")

func TestLivenessCheckAnswersOKWithoutAskingTheDatabase(t *testing.T) {
	t.Parallel()

	// arrange
	pinged := false
	handler := healthChecks(pingFunc(func(context.Context) error {
		pinged = true

		return errDatabaseDown
	}))

	// act
	response := get(t.Context(), handler, "/healthz")

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "ok\n", response.Body.String())
	assert.False(t, pinged)
}

func TestReadinessCheckAnswersOKWhenTheDatabaseAnswers(t *testing.T) {
	t.Parallel()

	// arrange
	handler := healthChecks(pingFunc(func(context.Context) error { return nil }))

	// act
	response := get(t.Context(), handler, "/readyz")

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "ok\n", response.Body.String())
}

func TestReadinessCheckAnswersServiceUnavailableWithoutTheErrorWhenTheDatabaseFails(t *testing.T) {
	t.Parallel()

	// arrange
	handler := healthChecks(pingFunc(func(context.Context) error { return errDatabaseDown }))

	// act
	response := get(loggingTo(t.Context(), io.Discard), handler, "/readyz")

	// assert
	assert.Equal(t, http.StatusServiceUnavailable, response.Code)
	assert.Equal(t, "the database doesn't answer\n", response.Body.String())
}

func TestReadinessCheckWaitsForTheDatabaseForASecond(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		deadline    time.Time
		hasDeadline bool
	)

	handler := healthChecks(pingFunc(func(ctx context.Context) error {
		deadline, hasDeadline = ctx.Deadline()

		return nil
	}))

	before := time.Now()

	// act
	get(t.Context(), handler, "/readyz")

	// assert
	after := time.Now()

	require.True(t, hasDeadline)
	assert.WithinRange(t, deadline, before.Add(time.Second), after.Add(time.Second))
}

func TestReadinessCheckPingsTheDatabaseWhenTheProberHasGivenUp(t *testing.T) {
	t.Parallel()

	// arrange
	var logs bytes.Buffer

	handler := healthChecks(pingFunc(func(ctx context.Context) error { return ctx.Err() }))
	ctx, cancel := context.WithCancel(loggingTo(t.Context(), &logs))
	cancel()

	// act
	response := get(ctx, handler, "/readyz")

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Empty(t, logs.String())
}

func TestReadinessCheckLogsOnlyWhenItsAnswerChanges(t *testing.T) {
	t.Parallel()

	// arrange
	var logs bytes.Buffer

	answers := []error{nil, errDatabaseDown, errDatabaseDown, nil, nil, errDatabaseDown}
	handler := healthChecks(pingFunc(func(context.Context) error {
		answer := answers[0]
		answers = answers[1:]

		return answer
	}))

	// act
	for range len(answers) {
		get(loggingTo(t.Context(), &logs), handler, "/readyz")
	}

	// assert
	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	require.Len(t, lines, 3)
	assert.Contains(t, lines[0], "level=WARN")
	assert.Contains(t, lines[0], "connection refused")
	assert.Contains(t, lines[1], "level=INFO")
	assert.Contains(t, lines[1], "the authority is ready again")
	assert.Contains(t, lines[2], "level=WARN")
}

// pingFunc is a database whose Ping calls the function.
type pingFunc func(ctx context.Context) error

func (f pingFunc) Ping(ctx context.Context) error {
	return f(ctx)
}

// healthChecks returns the metrics server's handler, whose /readyz pings
// database.
func healthChecks(database pinger) http.Handler {
	return metricsServer("", http.NotFoundHandler(), database).Handler
}

func loggingTo(ctx context.Context, w io.Writer) context.Context {
	return logging.NewContext(ctx, logging.New(w, slog.LevelInfo, logging.FormatText))
}

// get sends handler a GET request for path. The handler logs to the logger in
// ctx.
func get(ctx context.Context, handler http.Handler, path string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequestWithContext(ctx, http.MethodGet, path, nil))

	return response
}
