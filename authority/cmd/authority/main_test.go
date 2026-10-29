package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/internal/systemd"
)

func TestServeTellsSystemdWhenReadyAndStopping(t *testing.T) {
	// arrange
	socket := listenAsSystemd(t)
	notifier := openNotifier(t)
	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)

	// act
	go func() {
		served <- serve(ctx, notifier, map[string]*http.Server{"api": {Addr: "127.0.0.1:0"}}, shutdownTimeout)
	}()

	ready := receive(t, socket)

	cancel()

	stopping := receive(t, socket)

	// assert
	assert.Equal(t, "READY=1", ready)
	assert.Equal(t, "STOPPING=1", stopping)
	require.NoError(t, <-served)
}

func TestServeDoesNotTellSystemdReadyWhenListeningFails(t *testing.T) {
	// arrange
	socket := listenAsSystemd(t)
	notifier := openNotifier(t)

	taken, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = taken.Close() })

	// act
	err = serve(context.Background(), notifier, map[string]*http.Server{"api": {Addr: taken.Addr().String()}}, shutdownTimeout)

	// assert
	require.Error(t, err)
	require.NoError(t, socket.SetReadDeadline(time.Now().Add(100*time.Millisecond)))
	_, _, err = socket.ReadFrom(make([]byte, 64))
	assert.ErrorIs(t, err, os.ErrDeadlineExceeded)
}

func TestServeCancelsRequestsThatOutlastTheShutdownTimeout(t *testing.T) {
	// arrange
	socket := listenAsSystemd(t)
	notifier := openNotifier(t)
	ctx, cancel := context.WithCancel(context.Background())
	address := freeAddress(t)
	started := make(chan struct{})
	canceled := make(chan struct{})

	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(canceled)
	})

	served := make(chan error, 1)

	go func() {
		served <- serve(ctx, notifier, map[string]*http.Server{"api": {Addr: address, Handler: handler}}, 100*time.Millisecond)
	}()

	require.Equal(t, "READY=1", receive(t, socket))

	go func() {
		response, err := http.Get("http://" + address)
		if err == nil {
			_ = response.Body.Close()
		}
	}()

	<-started

	// act
	cancel()

	// assert
	select {
	case <-canceled:
	case <-time.After(5 * time.Second):
		t.Fatal("the request's context wasn't canceled")
	}

	require.ErrorIs(t, <-served, context.DeadlineExceeded)
}

// listenAsSystemd opens a socket that stands in for systemd's and points
// NOTIFY_SOCKET at it.
func listenAsSystemd(t *testing.T) net.PacketConn {
	t.Helper()

	// A path under t.TempDir contains the test's name and can exceed the
	// limit for Unix socket paths, which is 104 bytes on macOS.
	dir, err := os.MkdirTemp("", "notify")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	path := filepath.Join(dir, "socket")
	socket, err := net.ListenPacket("unixgram", path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = socket.Close() })

	t.Setenv("NOTIFY_SOCKET", path)

	return socket
}

func openNotifier(t *testing.T) *systemd.Notifier {
	t.Helper()

	notifier, err := systemd.Open()
	require.NoError(t, err)
	t.Cleanup(func() { _ = notifier.Close() })

	return notifier
}

// freeAddress returns a local address that nothing listens on right now.
func freeAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	address := listener.Addr().String()
	require.NoError(t, listener.Close())

	return address
}

func receive(t *testing.T, socket net.PacketConn) string {
	t.Helper()

	require.NoError(t, socket.SetReadDeadline(time.Now().Add(5*time.Second)))

	buffer := make([]byte, 64)
	n, _, err := socket.ReadFrom(buffer)
	require.NoError(t, err)

	return string(buffer[:n])
}
