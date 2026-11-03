package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/authority/internal/config"
)

func TestLoadCertificateReturnsNoneWhenTLSIsDisabled(t *testing.T) {
	t.Parallel()

	// act
	certificate, err := loadCertificate(config.TLS{Disabled: true})

	// assert
	require.NoError(t, err)
	assert.Nil(t, certificate)
}

func TestLoadCertificateLoadsTheFiles(t *testing.T) {
	t.Parallel()

	// arrange
	certFile, keyFile := writeKeyPair(t)

	// act
	certificate, err := loadCertificate(config.TLS{CertFile: certFile, KeyFile: keyFile})

	// assert
	require.NoError(t, err)
	assert.NotNil(t, certificate)
}

func TestLoadCertificateNamesTheMissingFileAndHowToServePlainHTTP(t *testing.T) {
	t.Parallel()

	// arrange
	missing := filepath.Join(t.TempDir(), "tls.crt")

	// act
	_, err := loadCertificate(config.TLS{CertFile: missing, KeyFile: missing})

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), missing)
	assert.Contains(t, err.Error(), "TIDEGATE_AUTHORITY__SERVER__TLS__DISABLED=true")
}

func writeKeyPair(t *testing.T) (certFile, keyFile string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{SerialNumber: big.NewInt(1), DNSNames: []string{"localhost"}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	directory := t.TempDir()
	certFile = filepath.Join(directory, "tls.crt")
	keyFile = filepath.Join(directory, "tls.key")

	require.NoError(t, os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))
	require.NoError(t, os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0o600))

	return certFile, keyFile
}
