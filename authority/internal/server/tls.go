package server

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/tidegate-dev/tidegate/authority/internal/config"
	"github.com/tidegate-dev/tidegate/internal/certreload"
)

// loadExternalCertificate returns nil when TLS is disabled.
func loadExternalCertificate(cfg config.ExternalTLS) (*certreload.Certificate, error) {
	if cfg.Disabled {
		return nil, nil
	}

	certificate, err := loadCertificate("external", cfg.CertFile, cfg.KeyFile)

	// Only a missing file can mean that a proxy terminates TLS. Any other
	// error needs fixing, not turning TLS off.
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w. To serve plain HTTP behind a proxy that terminates TLS, set %s=true",
			err, config.ExternalTLSDisabledVariable())
	}

	return certificate, err
}

func loadInternalCertificate(cfg config.InternalTLS) (*certreload.Certificate, error) {
	return loadCertificate("internal", cfg.CertFile, cfg.KeyFile)
}

func loadCertificate(apiName, certFile, keyFile string) (*certreload.Certificate, error) {
	certificate, err := certreload.Load(certFile, keyFile)

	if errors.Is(err, fs.ErrPermission) {
		return nil, fmt.Errorf("loading the %s API's TLS certificate: %w. "+
			"In Kubernetes, set the Pod's fsGroup to 65532, the user ID of the container image", apiName, err)
	}

	if err != nil {
		return nil, fmt.Errorf("loading the %s API's TLS certificate: %w", apiName, err)
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
