package server

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

	"github.com/phemera-dev/phemera/internal/systemd"
)

func TestServeTellsSystemdWhenReadyAndStopping(t *testing.T) {
	// arrange
	socket := listenAsSystemd(t)
	notifier := openNotifier(t)
	ctx, cancel := context.WithCancel(context.Background())
	server := &http.Server{ReadHeaderTimeout: time.Second}
	servers := map[string]listeningServer{"api": {server: server, listener: listenOnAnyPort(t)}}
	serveErrorChan := make(chan error, 1)

	// act
	go func() {
		serveErrorChan <- serve(ctx, notifier, servers, shutdownTimeout)
	}()

	ready := receive(t, socket)

	cancel()

	stopping := receive(t, socket)

	// assert
	assert.Equal(t, "READY=1", ready)
	assert.Equal(t, "STOPPING=1", stopping)
	require.NoError(t, <-serveErrorChan)
}

func TestListenFailsWhenAnAddressIsTaken(t *testing.T) {
	// arrange
	taken := listenOnAnyPort(t)

	// act
	_, err := listen(context.Background(), map[string]*http.Server{"api": {Addr: taken.Addr().String()}})

	// assert
	require.Error(t, err)
}

func TestServeCancelsRequestsThatOutlastTheShutdownTimeout(t *testing.T) {
	// arrange
	socket := listenAsSystemd(t)
	notifier := openNotifier(t)
	ctx, cancel := context.WithCancel(context.Background())
	listener := listenOnAnyPort(t)
	started := make(chan struct{})
	canceled := make(chan struct{})

	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(canceled)
	})

	serveErrorChan := make(chan error, 1)

	server := &http.Server{Handler: handler, ReadHeaderTimeout: time.Second}
	servers := map[string]listeningServer{"api": {server: server, listener: listener}}

	go func() {
		serveErrorChan <- serve(ctx, notifier, servers, 100*time.Millisecond)
	}()

	require.Equal(t, "READY=1", receive(t, socket))

	go func() {
		response, err := http.Get("http://" + listener.Addr().String())
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

	require.ErrorIs(t, <-serveErrorChan, context.DeadlineExceeded)
}

// listenAsSystemd opens a socket that stands in for systemd's and points
// NOTIFY_SOCKET at it.
func listenAsSystemd(t testing.TB) net.PacketConn {
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

func openNotifier(t testing.TB) *systemd.Notifier {
	t.Helper()

	notifier, err := systemd.Open()
	require.NoError(t, err)
	t.Cleanup(func() { _ = notifier.Close() })

	return notifier
}

func listenOnAnyPort(t testing.TB) net.Listener {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	return listener
}

func receive(t *testing.T, socket net.PacketConn) string {
	t.Helper()

	require.NoError(t, socket.SetReadDeadline(time.Now().Add(5*time.Second)))

	buffer := make([]byte, 64)
	n, _, err := socket.ReadFrom(buffer)
	require.NoError(t, err)

	return string(buffer[:n])
}
