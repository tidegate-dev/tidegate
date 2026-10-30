package certreload

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/tidegate-dev/tidegate/internal/logging"
)

func TestLoadRejectsAKeyOfAnotherCertificate(t *testing.T) {
	t.Parallel()

	// arrange
	dir := t.TempDir()
	certFile, _ := writeKeyPair(t, dir, 1)
	_, otherKeyFile := writeKeyPair(t, t.TempDir(), 2)

	// act
	_, err := Load(certFile, otherKeyFile)

	// assert
	assert.ErrorContains(t, err, "private key does not match public key")
}

func TestWatchPicksUpNewFiles(t *testing.T) {
	t.Parallel()

	// arrange
	dir := t.TempDir()
	certificate := load(t, dir, 1)
	watch(t, certificate, nil)

	// act
	writeKeyPair(t, dir, 2)

	// assert
	assertSerialBecomes(t, certificate, 2)
}

func TestWatchPicksUpASwappedSymlinkLikeAKubernetesSecret(t *testing.T) {
	t.Parallel()

	// arrange
	// Kubernetes writes a Secret's files to a new directory and then points
	// the ..data symlink at it.
	dir := t.TempDir()
	writeKeyPair(t, filepath.Join(dir, "..v1"), 1)
	require.NoError(t, os.Symlink("..v1", filepath.Join(dir, "..data")))
	require.NoError(t, os.Symlink(filepath.Join("..data", "cert.pem"), filepath.Join(dir, "cert.pem")))
	require.NoError(t, os.Symlink(filepath.Join("..data", "key.pem"), filepath.Join(dir, "key.pem")))

	certificate, err := Load(filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem"))
	require.NoError(t, err)

	certificate.checkInterval = time.Hour
	watch(t, certificate, nil)

	// act
	writeKeyPair(t, filepath.Join(dir, "..v2"), 2)
	require.NoError(t, os.Symlink("..v2", filepath.Join(dir, "..data_tmp")))
	require.NoError(t, os.Rename(filepath.Join(dir, "..data_tmp"), filepath.Join(dir, "..data")))

	// assert
	assertSerialBecomes(t, certificate, 2)
}

func TestWatchChecksTheFilesWhenWatchingThemFails(t *testing.T) {
	t.Parallel()

	// arrange
	dir := t.TempDir()
	certificate := load(t, dir, 1)
	certificate.newWatcher = func() (*fsnotify.Watcher, error) {
		return nil, errors.New("no watcher in this test")
	}
	certificate.checkInterval = 50 * time.Millisecond
	watch(t, certificate, nil)

	// act
	writeKeyPair(t, dir, 2)

	// assert
	assertSerialBecomes(t, certificate, 2)
}

func TestWatchKeepsTheCertificateWhenTheNewFilesAreBroken(t *testing.T) {
	t.Parallel()

	// arrange
	dir := t.TempDir()
	certificate := load(t, dir, 1)
	logs := &syncBuffer{}
	watch(t, certificate, logs)

	// act
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cert.pem"), []byte("not a certificate"), 0o600))

	// assert
	assert.Eventually(t, func() bool {
		return bytes.Contains(logs.Bytes(), []byte("keeping the TLS certificate"))
	}, 5*time.Second, 10*time.Millisecond)
	assert.Equal(t, int64(1), serial(t, certificate))
}

func TestWatchLogsBrokenFilesOnce(t *testing.T) {
	t.Parallel()

	// arrange
	dir := t.TempDir()
	certificate := load(t, dir, 1)
	logs := &syncBuffer{}
	watch(t, certificate, logs)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cert.pem"), []byte("not a certificate"), 0o600))
	assert.Eventually(t, func() bool {
		return bytes.Contains(logs.Bytes(), []byte("keeping the TLS certificate"))
	}, 5*time.Second, 10*time.Millisecond)

	// act
	require.NoError(t, os.WriteFile(filepath.Join(dir, "unrelated"), nil, 0o600))
	time.Sleep(100 * time.Millisecond)

	// assert
	assert.Equal(t, 1, bytes.Count(logs.Bytes(), []byte("keeping the TLS certificate")))
}

func TestWatchReportsWhenTheCertificateExpires(t *testing.T) {
	// The meter provider is global, so this test can't run in parallel.

	// arrange
	reader := sdkmetric.NewManualReader()
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))

	dir := t.TempDir()
	certificate := load(t, dir, 1)
	watch(t, certificate, nil)

	var metrics metricdata.ResourceMetrics

	// act
	assert.Eventually(t, func() bool {
		require.NoError(t, reader.Collect(context.Background(), &metrics))

		return len(metrics.ScopeMetrics) > 0
	}, 5*time.Second, 10*time.Millisecond)

	// assert
	require.Len(t, metrics.ScopeMetrics, 1)
	require.Len(t, metrics.ScopeMetrics[0].Metrics, 1)

	expiry := metrics.ScopeMetrics[0].Metrics[0]
	assert.Equal(t, "tidegate.tls.certificate.expiry", expiry.Name)
	assert.Equal(t, "s", expiry.Unit)

	points := expiry.Data.(metricdata.Gauge[int64]).DataPoints
	require.Len(t, points, 1)
	assert.Equal(t, notAfter(1).Unix(), points[0].Value)
}

func load(t *testing.T, dir string, serial int64) *Certificate {
	t.Helper()

	certFile, keyFile := writeKeyPair(t, dir, serial)
	certificate, err := Load(certFile, keyFile)
	require.NoError(t, err)

	// Only the test of the timer wants it to fire.
	certificate.checkInterval = time.Hour

	return certificate
}

// watch runs certificate.Watch until the test ends and returns once it
// watches the files. It logs to logs if it isn't nil.
func watch(t *testing.T, certificate *Certificate, logs *syncBuffer) {
	t.Helper()

	certificate.settleDelay = 10 * time.Millisecond

	var logger *slog.Logger
	if logs != nil {
		logger = slog.New(slog.NewTextHandler(logs, nil))
	} else {
		logger = slog.New(slog.DiscardHandler)
	}

	ctx, cancel := context.WithCancel(logging.NewContext(context.Background(), logger))
	started := make(chan struct{})
	done := make(chan struct{})

	certificate.started = func() { close(started) }

	go func() {
		certificate.Watch(ctx)
		close(done)
	}()

	t.Cleanup(func() {
		cancel()
		<-done
	})

	<-started
}

func assertSerialBecomes(t *testing.T, certificate *Certificate, want int64) {
	t.Helper()

	assert.Eventually(t, func() bool {
		return serial(t, certificate) == want
	}, 5*time.Second, 10*time.Millisecond)
}

func serial(t *testing.T, certificate *Certificate) int64 {
	t.Helper()

	got, err := certificate.Get(nil)
	require.NoError(t, err)

	return got.Leaf.SerialNumber.Int64()
}

// notAfter gives each serial number its own expiry, so a test can tell the
// certificates apart by it.
func notAfter(serial int64) time.Time {
	return time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, int(serial))
}

// writeKeyPair writes a self-signed certificate with serial and its key to
// cert.pem and key.pem in dir. It writes the key first, so a watcher that
// reads the files in between sees a key that doesn't match.
func writeKeyPair(t *testing.T, dir string, serial int64) (certFile, keyFile string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(dir, 0o700))

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		DNSNames:     []string{"localhost"},
		NotAfter:     notAfter(serial),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	certFile = filepath.Join(dir, "cert.pem")
	keyFile = filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0o600))
	require.NoError(t, os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))

	return certFile, keyFile
}

// syncBuffer is a bytes.Buffer that the watcher can write to while the test
// reads it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

func (b *syncBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	return bytes.Clone(b.buf.Bytes())
}
