package db

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFallsBackToPlaintext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		url  string
		want bool
	}{
		{url: "postgres://phemera@host/phemera", want: true},
		{url: "host=host user=phemera", want: true},
		{url: "postgres://phemera@host/phemera?sslmode=prefer", want: true},
		{url: "postgres://phemera@host/phemera?sslmode=disable", want: false},
		{url: "postgres://phemera@host/phemera?sslmode=require", want: false},
		{url: "postgres://phemera@host/phemera?sslmode=verify-full", want: false},
		{url: "postgres://phemera@host1,host2/phemera?sslmode=verify-full", want: false},
	}

	for _, test := range tests {
		t.Run(test.url, func(t *testing.T) {
			t.Parallel()

			// arrange
			config, err := pgxpool.ParseConfig(test.url)
			require.NoError(t, err)

			// act
			got := fallsBackToPlaintext(config)

			// assert
			assert.Equal(t, test.want, got)
		})
	}
}

func TestFallsBackToPlaintextFollowsPGSSLMODE(t *testing.T) {
	// arrange
	t.Setenv("PGSSLMODE", "disable")

	config, err := pgxpool.ParseConfig("postgres://phemera@host/phemera")
	require.NoError(t, err)

	// act
	got := fallsBackToPlaintext(config)

	// assert
	assert.False(t, got)
}

func TestOpenRejectsAnInvalidURLWithoutTheURL(t *testing.T) {
	t.Parallel()

	// act
	// pgx can't find the password to mask in this URL, because the @ is missing.
	_, err := Open(t.Context(), "postgres://phemera:hunter2host:x/phemera")

	// assert
	assert.EqualError(t, err, "the database URL isn't a valid Postgres connection URL, or a PG variable like PGSSLMODE is invalid")
}

func TestOpenNamesAMissingTLSFile(t *testing.T) {
	t.Parallel()

	// arrange
	missing := filepath.Join(t.TempDir(), "ca.crt")

	// act
	_, err := Open(t.Context(), "postgres://phemera@host/phemera?sslmode=verify-full&sslrootcert="+missing)

	// assert
	assert.EqualError(t, err, "the database URL can't be used, "+
		"because sslrootcert or PGSSLROOTCERT names a file that can't be read: no such file or directory")
}

func TestOpenWhenAvailableFailsAtOnceWhenTheServerRefusesTheLogin(t *testing.T) {
	t.Parallel()

	// arrange
	address := answerEveryLogin(t, "28P01")

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	// act
	_, err := OpenWhenAvailable(ctx, "postgres://phemera@"+address+"/phemera?sslmode=disable")

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SQLSTATE 28P01")
	assert.NoError(t, ctx.Err(), "it waited")
}

// The tests that wait run in a synctest bubble, where time only passes when
// every goroutine in it waits. Their warnings go to the default logger, which
// is global, so they don't run in parallel.

func TestWaitUntilAvailableTriesAgainUntilThePingSucceeds(t *testing.T) {
	refused := refusedError(t)

	synctest.Test(t, func(t *testing.T) {
		// arrange
		logs := captureLogs(t)
		pings := 0
		ping := func(context.Context) error {
			pings++
			if pings <= 3 {
				return refused
			}

			return nil
		}
		start := time.Now()

		// act
		err := waitUntilAvailable(t.Context(), ping)

		// assert
		require.NoError(t, err)
		assert.Equal(t, 4, pings)
		assert.Equal(t, (1+2+4)*time.Second, time.Since(start))
		assert.Equal(t, 3, strings.Count(logs.String(), "level=WARN"))
		assert.Contains(t, logs.String(), "connection refused")
	})
}

func TestWaitUntilAvailableWaitsAtMostTheLongestRetryDelay(t *testing.T) {
	refused := refusedError(t)

	synctest.Test(t, func(t *testing.T) {
		// arrange
		captureLogs(t)

		var delays []time.Duration

		last := time.Now()
		ping := func(context.Context) error {
			delays = append(delays, time.Since(last))
			last = time.Now()

			if len(delays) <= 6 {
				return refused
			}

			return nil
		}

		// act
		err := waitUntilAvailable(t.Context(), ping)

		// assert
		require.NoError(t, err)
		assert.Equal(t, []time.Duration{0, 1 * time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 10 * time.Second, 10 * time.Second}, delays)
	})
}

func TestWaitUntilAvailableFailsAtOnceWhenWaitingDoesNotHelp(t *testing.T) {
	refused := passwordError(t)

	synctest.Test(t, func(t *testing.T) {
		// arrange
		captureLogs(t)

		ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
		defer cancel()

		pings := 0
		ping := func(context.Context) error {
			pings++

			return refused
		}
		start := time.Now()

		// act
		err := waitUntilAvailable(ctx, ping)

		// assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SQLSTATE 28P01")
		assert.Equal(t, 1, pings)
		assert.Zero(t, time.Since(start))
	})
}

func TestWaitUntilAvailableStopsAtOnceWhenTheContextEndsWhileItWaits(t *testing.T) {
	refused := refusedError(t)

	synctest.Test(t, func(t *testing.T) {
		// arrange
		captureLogs(t)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		ping := func(context.Context) error { return refused }
		start := time.Now()

		// The tries are at 0, 1, 3, 7, 15 and 25 seconds, the next would be at 35.
		time.AfterFunc(30*time.Second, cancel)

		// act
		err := waitUntilAvailable(ctx, ping)

		// assert
		require.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, 30*time.Second, time.Since(start))
	})
}

func TestUnavailable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  func(t *testing.T) error
		want bool
	}{
		{name: "nothing listens", err: refusedError, want: true},
		{name: "nothing listens, sslmode=prefer", err: func(t *testing.T) error {
			return openError(t, "postgres://phemera@"+closedAddress(t)+"/phemera?sslmode=prefer")
		}, want: true},
		{name: "the host doesn't resolve", err: func(*testing.T) error {
			return fmt.Errorf("hostname resolving error: %w", &net.DNSError{Err: "no such host", Name: "db.example.com", IsNotFound: true})
		}, want: true},
		{name: "the server doesn't answer within connect_timeout", err: func(t *testing.T) error {
			return openError(t, "postgres://phemera@"+answerNothing(t)+"/phemera?sslmode=disable&connect_timeout=1")
		}, want: true},
		{name: "the server is starting", err: func(t *testing.T) error {
			return openError(t, "postgres://phemera@"+answerEveryLogin(t, "57P03")+"/phemera?sslmode=disable")
		}, want: true},
		{name: "the server is starting and refuses TLS, sslmode=prefer", err: func(t *testing.T) error {
			return openError(t, "postgres://phemera@"+answerEveryLogin(t, "57P03")+"/phemera?sslmode=prefer")
		}, want: true},
		{name: "the password is wrong", err: passwordError, want: false},
		{name: "the password is wrong and the server refuses TLS, sslmode=prefer", err: func(t *testing.T) error {
			return openError(t, "postgres://phemera@"+answerEveryLogin(t, "28P01")+"/phemera?sslmode=prefer")
		}, want: false},
		{name: "the database doesn't exist", err: func(t *testing.T) error {
			return openError(t, "postgres://phemera@"+answerEveryLogin(t, "3D000")+"/phemera?sslmode=disable")
		}, want: false},
		{name: "the server has too many connections", err: func(t *testing.T) error {
			return openError(t, "postgres://phemera@"+answerEveryLogin(t, "53300")+"/phemera?sslmode=disable")
		}, want: false},
		{name: "the URL is invalid", err: func(t *testing.T) error {
			return openError(t, "postgres://phemera:hunter2host:x/phemera")
		}, want: false},
		{name: "the schema is too old", err: func(*testing.T) error { return ErrSchemaTooOld }, want: false},
		{name: "the context was canceled", err: func(*testing.T) error { return context.Canceled }, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			// arrange
			err := test.err(t)

			// act
			got := unavailable(err)

			// assert
			assert.Equal(t, test.want, got, err.Error())
		})
	}
}

// openError returns the error that Open returns for url.
func openError(t *testing.T, url string) error {
	t.Helper()

	pool, err := Open(t.Context(), url)
	if err == nil {
		pool.Close()
	}

	require.Error(t, err)

	return err
}

func refusedError(t *testing.T) error {
	t.Helper()

	return openError(t, "postgres://phemera@"+closedAddress(t)+"/phemera?sslmode=disable")
}

func passwordError(t *testing.T) error {
	t.Helper()

	return openError(t, "postgres://phemera@"+answerEveryLogin(t, "28P01")+"/phemera?sslmode=disable")
}

// closedAddress returns a local address that nothing listens on.
func closedAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	require.NoError(t, listener.Close())

	return listener.Addr().String()
}

// answerEveryLogin returns the address of a server that refuses TLS and
// answers every login with an error with code, like Postgres does when it
// refuses a login or is still starting.
func answerEveryLogin(t *testing.T, code string) string {
	t.Helper()

	return listen(t, func(connection net.Conn) {
		backend := pgproto3.NewBackend(connection, connection)

		for {
			message, err := backend.ReceiveStartupMessage()
			if err != nil {
				return
			}

			_, isSSLRequest := message.(*pgproto3.SSLRequest)
			if !isSSLRequest {
				backend.Send(&pgproto3.ErrorResponse{Severity: "FATAL", Code: code, Message: "refused by the test"})
				_ = backend.Flush()

				return
			}

			_, err = connection.Write([]byte("N"))
			if err != nil {
				return
			}
		}
	})
}

// answerNothing returns the address of a server that accepts connections
// and never answers.
func answerNothing(t *testing.T) string {
	t.Helper()

	return listen(t, func(connection net.Conn) {
		_, _ = io.Copy(io.Discard, connection)
	})
}

// listen calls handle with each connection to the address it returns, and
// closes the connection afterwards.
func listen(t *testing.T, handle func(net.Conn)) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}

			go func() {
				defer func() { _ = connection.Close() }()

				handle(connection)
			}()
		}
	}()

	return listener.Addr().String()
}

// captureLogs points the default logger at the buffer it returns until the
// test ends.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	var logs bytes.Buffer

	previous := slog.Default()

	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	return &logs
}
