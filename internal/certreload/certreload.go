// Package certreload serves a TLS certificate from files and picks up new
// files without a restart. It needs Unix to check who owns the key file.
package certreload

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/tidegate-dev/tidegate/internal/logging"
)

// ErrKeyTooShort means the certificate's key is shorter than NIST allows
// through 2030.
var ErrKeyTooShort = errors.New("the TLS certificate's key is too short")

// The minimums of NIST SP 800-57 Part 1 for use through 2030, 112 bits of
// security.
const (
	minRSABits   = 2048
	minCurveBits = 224
)

// Certificate is a TLS certificate and its key, read from two files.
type Certificate struct {
	certFile string
	keyFile  string
	current  atomic.Pointer[keyPair]

	// fsnotify misses some changes, and watching can fail, so Watch also
	// checks the files on this interval, like the Kubernetes API server.
	checkInterval time.Duration
	// Tools write the certificate and the key one after the other, so
	// Watch waits this long after a change before it reads them.
	settleDelay time.Duration
	newWatcher  func() (*fsnotify.Watcher, error)
	// started lets tests wait until Watch watches the files.
	started func()

	// lastError is the error Watch logged last, so it logs each error once
	// and not on every check.
	lastError string
}

type keyPair struct {
	hashes      fileHashes
	certificate *tls.Certificate
}

// fileHashes are the SHA-256 hashes of the certificate's file and the key's
// file, to tell whether the files changed.
type fileHashes [2][sha256.Size]byte

// Load reads the certificate from certFile and its key from keyFile.
func Load(certFile, keyFile string) (*Certificate, error) {
	c := &Certificate{
		certFile:      certFile,
		keyFile:       keyFile,
		checkInterval: time.Minute,
		settleDelay:   time.Second,
		newWatcher:    fsnotify.NewWatcher,
		started:       func() {},
	}

	pair, err := c.readKeyPair()
	if err != nil {
		return nil, err
	}

	c.current.Store(pair)

	return c, nil
}

// Get returns the certificate in use. It has the signature of
// tls.Config.GetCertificate.
func (c *Certificate) Get(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return c.current.Load().certificate, nil
}

// Watch picks up new files until ctx is done. It switches only when the new
// files hold a certificate and its key, and otherwise keeps the one in use.
func (c *Certificate) Watch(ctx context.Context) {
	watcher := &dirWatcher{
		dirs:       []string{filepath.Dir(c.certFile), filepath.Dir(c.keyFile)},
		newWatcher: c.newWatcher,
	}
	watcher.start(ctx)

	defer watcher.close()

	ticker := time.NewTicker(c.checkInterval)
	defer ticker.Stop()

	var settled <-chan time.Time // nil until a file changes

	// The files may have changed since Load read them.
	c.reload(ctx)
	c.started()

	for {
		select {
		case <-ctx.Done():
			return
		case <-watcher.events():
			settled = time.After(c.settleDelay)
		case err := <-watcher.errors():
			watcher.stop(ctx, err)
		case <-settled:
			settled = nil

			c.reload(ctx)
		case <-ticker.C:
			watcher.start(ctx)
			c.reload(ctx)
		}
	}
}

// ReportExpiry reports on the global meter provider when the certificate in
// use expires.
func (c *Certificate) ReportExpiry() (metric.Registration, error) {
	meter := otel.Meter("github.com/tidegate-dev/tidegate/internal/certreload")

	expiry, err := meter.Int64ObservableGauge("tidegate.tls.certificate.expiry",
		metric.WithUnit("s"),
		metric.WithDescription("When the TLS certificate in use expires, in seconds since 1970"))
	if err != nil {
		return nil, fmt.Errorf("creating the gauge for the TLS certificate's expiry: %w", err)
	}

	attributes := metric.WithAttributes(attribute.String("cert_file", c.certFile))

	return meter.RegisterCallback(func(_ context.Context, observer metric.Observer) error {
		observer.ObserveInt64(expiry, c.current.Load().certificate.Leaf.NotAfter.Unix(), attributes)

		return nil
	}, expiry)
}

func (c *Certificate) reload(ctx context.Context) {
	pair, err := c.readKeyPair()
	if err != nil {
		c.logOnce(ctx, err)

		return
	}

	c.lastError = ""

	if pair.hashes == c.current.Load().hashes {
		return
	}

	c.current.Store(pair)
	logging.FromContext(ctx).Info("using a new TLS certificate", "cert_file", c.certFile, "expires", pair.certificate.Leaf.NotAfter)
}

func (c *Certificate) logOnce(ctx context.Context, err error) {
	if err.Error() == c.lastError {
		return
	}

	c.lastError = err.Error()
	logging.FromContext(ctx).Error("keeping the TLS certificate in use, because its new files can't be used",
		"cert_file", c.certFile, "error", err)
}

func (c *Certificate) readKeyPair() (*keyPair, error) {
	certPEM, err := os.ReadFile(c.certFile)
	if err != nil {
		return nil, err
	}

	keyPEM, err := readKeyFile(c.keyFile)
	if err != nil {
		return nil, err
	}

	certificate, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	err = checkKeyLength(certificate.Leaf)
	if err != nil {
		return nil, err
	}

	return &keyPair{
		hashes:      fileHashes{sha256.Sum256(certPEM), sha256.Sum256(keyPEM)},
		certificate: &certificate,
	}, nil
}

func checkKeyLength(leaf *x509.Certificate) error {
	switch key := leaf.PublicKey.(type) {
	case *rsa.PublicKey:
		if bits := key.N.BitLen(); bits < minRSABits {
			return fmt.Errorf("%w: its RSA key has %d bits, and tidegate needs at least %d", ErrKeyTooShort, bits, minRSABits)
		}
	case *ecdsa.PublicKey:
		if bits := key.Curve.Params().BitSize; bits < minCurveBits {
			return fmt.Errorf("%w: its elliptic curve has %d bits, and tidegate needs at least %d", ErrKeyTooShort, bits, minCurveBits)
		}
	}

	return nil
}
