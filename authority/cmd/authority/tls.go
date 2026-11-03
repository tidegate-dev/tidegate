package main

import (
	"fmt"

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
	if err != nil {
		return nil, fmt.Errorf("loading the TLS certificate: %w. "+
			"To serve plain HTTP behind a proxy that terminates TLS, set TIDEGATE_AUTHORITY__SERVER__TLS__DISABLED=true", err)
	}

	return certificate, nil
}
