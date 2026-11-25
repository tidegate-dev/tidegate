package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phemera-dev/phemera/authority/internal/config"
)

func TestLoadExternalCertificateReturnsNoneWhenTLSIsDisabled(t *testing.T) {
	t.Parallel()

	// act
	certificate, err := loadExternalCertificate(config.ExternalTLS{Disabled: true})

	// assert
	require.NoError(t, err)
	assert.Nil(t, certificate)
}

func TestLoadExternalCertificateLoadsTheFiles(t *testing.T) {
	t.Parallel()

	// arrange
	certFile, keyFile := writeKeyPair(t)

	// act
	certificate, err := loadExternalCertificate(config.ExternalTLS{CertFile: certFile, KeyFile: keyFile})

	// assert
	require.NoError(t, err)
	assert.NotNil(t, certificate)
}

func TestLoadExternalCertificateNamesTheMissingKeyAndHowToServePlainHTTP(t *testing.T) {
	t.Parallel()

	// arrange
	certFile, _ := writeKeyPair(t)
	missingKey := filepath.Join(t.TempDir(), "tls.key")

	// act
	_, err := loadExternalCertificate(config.ExternalTLS{CertFile: certFile, KeyFile: missingKey})

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "external API")
	assert.Contains(t, err.Error(), missingKey)
	assert.Contains(t, err.Error(), config.ExternalTLSDisabledVariable()+"=true")
}

func TestLoadInternalCertificateLoadsTheFiles(t *testing.T) {
	t.Parallel()

	// arrange
	certFile, keyFile := writeKeyPair(t)

	// act
	certificate, err := loadInternalCertificate(config.InternalTLS{CertFile: certFile, KeyFile: keyFile})

	// assert
	require.NoError(t, err)
	assert.NotNil(t, certificate)
}

func TestLoadInternalCertificateNamesTheMissingKeyWithoutSuggestingPlainHTTP(t *testing.T) {
	t.Parallel()

	// arrange
	certFile, _ := writeKeyPair(t)
	missingKey := filepath.Join(t.TempDir(), "tls.key")

	// act
	_, err := loadInternalCertificate(config.InternalTLS{CertFile: certFile, KeyFile: missingKey})

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "internal API")
	assert.Contains(t, err.Error(), missingKey)
	assert.NotContains(t, err.Error(), "plain HTTP")
}

func TestLoadExternalCertificatePointsToTheKeyRulesWhenItCantReadTheKey(t *testing.T) {
	t.Parallel()

	if os.Geteuid() == 0 {
		t.Skip("root reads the key whatever its mode")
	}

	// arrange
	certFile, keyFile := writeKeyPair(t)
	require.NoError(t, os.Chmod(keyFile, 0o000))

	// act
	_, err := loadExternalCertificate(config.ExternalTLS{CertFile: certFile, KeyFile: keyFile})

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fsGroup")
}

func TestLoadExternalCertificateDoesntSuggestPlainHTTPForAnInvalidCertificate(t *testing.T) {
	t.Parallel()

	// arrange
	certFile, keyFile := writeKeyPair(t)
	require.NoError(t, os.WriteFile(certFile, []byte("not a certificate"), 0o600))

	// act
	_, err := loadExternalCertificate(config.ExternalTLS{CertFile: certFile, KeyFile: keyFile})

	// assert
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "plain HTTP")
}

func writeKeyPair(t testing.TB) (certFile, keyFile string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
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
