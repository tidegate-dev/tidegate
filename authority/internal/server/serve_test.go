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
	serveErrorChan := make(chan error, 1)

	// act
	go func() {
		serveErrorChan <- serve(ctx, notifier, map[string]*http.Server{"api": {Addr: addressWithAnyPort}}, shutdownTimeout, ignoreAddresses)
	}()

	ready := receive(t, socket)

	cancel()

	stopping := receive(t, socket)

	// assert
	assert.Equal(t, "READY=1", ready)
	assert.Equal(t, "STOPPING=1", stopping)
	require.NoError(t, <-serveErrorChan)
}

func TestServeDoesNotTellSystemdReadyWhenListeningFails(t *testing.T) {
	// arrange
	socket := listenAsSystemd(t)
	notifier := openNotifier(t)

	taken, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = taken.Close() })

	// act
	err = serve(context.Background(), notifier, map[string]*http.Server{"api": {Addr: taken.Addr().String()}}, shutdownTimeout, ignoreAddresses)

	// assert
	require.Error(t, err)
	require.NoError(t, socket.SetReadDeadline(time.Now().Add(100*time.Millisecond)))
	_, _, err = socket.ReadFrom(make([]byte, 64))
	assert.ErrorIs(t, err, os.ErrDeadlineExceeded)
}

func TestServeReportsTheAddressThatAServerListensOn(t *testing.T) {
	// arrange
	notifier := openNotifier(t)
	addressesChan := make(chan map[string]string, 1)
	serveErrorChan := make(chan error, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// act
	go func() {
		serveErrorChan <- serve(ctx, notifier, map[string]*http.Server{"api": {Addr: addressWithAnyPort}}, shutdownTimeout, sendTo(addressesChan))
	}()

	// assert
	select {
	case addresses := <-addressesChan:
		conn, err := net.Dial("tcp", addresses["api"])
		require.NoError(t, err)
		require.NoError(t, conn.Close())
	case err := <-serveErrorChan:
		t.Fatalf("serve stopped before it reported an address: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("serve didn't report an address")
	}

	cancel()
	require.NoError(t, <-serveErrorChan)
}

func TestServeCancelsRequestsThatOutlastTheShutdownTimeout(t *testing.T) {
	// arrange
	socket := listenAsSystemd(t)
	notifier := openNotifier(t)
	ctx, cancel := context.WithCancel(context.Background())
	addressesChan := make(chan map[string]string, 1)
	started := make(chan struct{})
	canceled := make(chan struct{})

	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(canceled)
	})

	serveErrorChan := make(chan error, 1)

	go func() {
		servers := map[string]*http.Server{"api": {Addr: addressWithAnyPort, Handler: handler}}
		serveErrorChan <- serve(ctx, notifier, servers, 100*time.Millisecond, sendTo(addressesChan))
	}()

	require.Equal(t, "READY=1", receive(t, socket))

	address := (<-addressesChan)["api"]

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

	require.ErrorIs(t, <-serveErrorChan, context.DeadlineExceeded)
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

// The system picks a free port for a listener on port 0.
const addressWithAnyPort = "127.0.0.1:0"

func ignoreAddresses(map[string]string) {}

func sendTo(addressesChan chan<- map[string]string) func(map[string]string) {
	return func(addresses map[string]string) {
		addressesChan <- addresses
	}
}

func receive(t *testing.T, socket net.PacketConn) string {
	t.Helper()

	require.NoError(t, socket.SetReadDeadline(time.Now().Add(5*time.Second)))

	buffer := make([]byte, 64)
	n, _, err := socket.ReadFrom(buffer)
	require.NoError(t, err)

	return string(buffer[:n])
}
