// Package certreload serves a TLS certificate from files and picks up new
// files without a restart.
package certreload

import (
	"bytes"
	"context"
	"crypto/tls"
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

	// broken holds the files that Watch last failed to load, so it logs
	// them once and not on every check.
	broken *brokenFiles
}

type keyPair struct {
	certPEM     []byte
	keyPEM      []byte
	certificate *tls.Certificate
}

type brokenFiles struct {
	certPEM []byte
	keyPEM  []byte
	err     string
}

func (b *brokenFiles) equal(other *brokenFiles) bool {
	return b != nil && other != nil && bytes.Equal(b.certPEM, other.certPEM) &&
		bytes.Equal(b.keyPEM, other.keyPEM) && b.err == other.err
}

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

	certPEM, keyPEM, err := c.readFiles()
	if err != nil {
		return nil, err
	}

	pair, err := newKeyPair(certPEM, keyPEM)
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
// It also reports on the global meter provider when the certificate in use
// expires.
func (c *Certificate) Watch(ctx context.Context) {
	logger := logging.FromContext(ctx)

	registration, err := c.registerExpiry()
	if err != nil {
		logger.Error("reporting when the TLS certificate expires", "cert_file", c.certFile, "error", err)
	} else {
		defer func() {
			_ = registration.Unregister()
		}()
	}

	ticker := time.NewTicker(c.checkInterval)
	defer ticker.Stop()

	settle := time.NewTimer(0)
	<-settle.C

	watcher := c.watch(ctx)

	defer func() {
		if watcher != nil {
			_ = watcher.Close()
		}
	}()

	// The files may have changed since Load read them.
	c.reload(ctx)
	c.started()

	for {
		var (
			events <-chan fsnotify.Event
			errs   <-chan error
		)

		if watcher != nil {
			events, errs = watcher.Events, watcher.Errors
		}

		select {
		case <-ctx.Done():
			return
		case <-events:
			settle.Reset(c.settleDelay)
		case err := <-errs:
			logger.Error("watching the TLS certificate's files, checking them every minute instead", "cert_file", c.certFile, "error", err)

			_ = watcher.Close()
			watcher = nil
		case <-settle.C:
			c.reload(ctx)
		case <-ticker.C:
			if watcher == nil {
				watcher = c.watch(ctx)
			}

			c.reload(ctx)
		}
	}
}

// watch watches the directories of both files, because Kubernetes replaces
// a Secret's files by swapping a symlink next to them. It returns nil if
// that fails.
func (c *Certificate) watch(ctx context.Context) *fsnotify.Watcher {
	watcher, err := c.newWatcher()
	if err == nil {
		for _, dir := range []string{filepath.Dir(c.certFile), filepath.Dir(c.keyFile)} {
			err = watcher.Add(dir)
			if err != nil {
				_ = watcher.Close()
				break
			}
		}
	}

	if err != nil {
		logging.FromContext(ctx).Error("watching the TLS certificate's files, checking them every minute instead",
			"cert_file", c.certFile, "error", err)

		return nil
	}

	return watcher
}

func (c *Certificate) reload(ctx context.Context) {
	certPEM, keyPEM, err := c.readFiles()

	var pair *keyPair

	if err == nil {
		current := c.current.Load()
		if bytes.Equal(certPEM, current.certPEM) && bytes.Equal(keyPEM, current.keyPEM) {
			c.broken = nil

			return
		}

		pair, err = newKeyPair(certPEM, keyPEM)
	}

	if err != nil {
		broken := &brokenFiles{certPEM: certPEM, keyPEM: keyPEM, err: err.Error()}
		if !broken.equal(c.broken) {
			logging.FromContext(ctx).Error("keeping the TLS certificate in use, because its files changed and don't hold a certificate and its key",
				"cert_file", c.certFile, "error", err)
		}

		c.broken = broken

		return
	}

	c.broken = nil
	c.current.Store(pair)
	logging.FromContext(ctx).Info("using a new TLS certificate", "cert_file", c.certFile, "expires", pair.certificate.Leaf.NotAfter)
}

func (c *Certificate) readFiles() (certPEM, keyPEM []byte, err error) {
	certPEM, err = os.ReadFile(c.certFile)
	if err != nil {
		return nil, nil, err
	}

	keyPEM, err = os.ReadFile(c.keyFile)
	if err != nil {
		return nil, nil, err
	}

	return certPEM, keyPEM, nil
}

func newKeyPair(certPEM, keyPEM []byte) (*keyPair, error) {
	certificate, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	return &keyPair{certPEM: certPEM, keyPEM: keyPEM, certificate: &certificate}, nil
}

func (c *Certificate) registerExpiry() (metric.Registration, error) {
	meter := otel.Meter("github.com/tidegate-dev/tidegate/internal/certreload")

	expiry, err := meter.Int64ObservableGauge("tidegate.tls.certificate.expiry",
		metric.WithUnit("s"),
		metric.WithDescription("When the TLS certificate in use expires, in seconds since 1970"))
	if err != nil {
		return nil, fmt.Errorf("creating the gauge: %w", err)
	}

	attributes := metric.WithAttributes(attribute.String("cert_file", c.certFile))

	return meter.RegisterCallback(func(_ context.Context, observer metric.Observer) error {
		observer.ObserveInt64(expiry, c.current.Load().certificate.Leaf.NotAfter.Unix(), attributes)

		return nil
	}, expiry)
}
