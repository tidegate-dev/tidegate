package server

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/tidegate-dev/tidegate/authority/internal/config"
	"github.com/tidegate-dev/tidegate/internal/certreload"
)

// loadCertificate loads the API server's certificate. It returns nil when TLS
// is disabled.
func loadCertificate(cfg config.TLS) (*certreload.Certificate, error) {
	if cfg.Disabled {
		return nil, nil
	}

	certificate, err := certreload.Load(cfg.CertFile, cfg.KeyFile)

	// Only a missing file can mean that a proxy terminates TLS. Any other
	// error needs fixing, not turning TLS off.
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("loading the TLS certificate: %w. "+
			"To serve plain HTTP behind a proxy that terminates TLS, set TIDEGATE_AUTHORITY__SERVER__TLS__DISABLED=true", err)
	}

	if errors.Is(err, fs.ErrPermission) {
		return nil, fmt.Errorf("loading the TLS certificate: %w. "+
			"In Kubernetes, set the Pod's fsGroup to 65532, the user ID of the container image", err)
	}

	if err != nil {
		return nil, fmt.Errorf("loading the TLS certificate: %w", err)
	}

	return certificate, nil
}

// watchCertificate reports certificate's expiry as a metric and watches
// its files until ctx ends. The returned function stops the reporting. It
// does nothing if certificate is nil.
func watchCertificate(ctx context.Context, certificate *certreload.Certificate) (func(), error) {
	if certificate == nil {
		return func() {}, nil
	}

	registration, err := certificate.ReportExpiry()
	if err != nil {
		return nil, err
	}

	go certificate.Watch(ctx)

	return func() {
		_ = registration.Unregister()
	}, nil
}
