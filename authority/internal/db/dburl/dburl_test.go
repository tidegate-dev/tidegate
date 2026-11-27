package dburl_test

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

	"github.com/phemera-dev/phemera/authority/internal/db/dburl"
)

const (
	password   = "hunter2"
	host       = "db.example.com"
	baseURL    = "postgres://phemera:" + password + "@" + host + "/phemera"
	url        = baseURL + "?sslmode=verify-full"
	invalidURL = "isn't a valid Postgres connection URL, or a PG variable like PGSSLMODE is invalid"
)

// The PG variables make the subtests share the environment, so they can't run
// in parallel.
func TestParse(t *testing.T) {
	// arrange
	certFile, keyFile := writeKeyPair(t)
	missing := filepath.Join(t.TempDir(), "missing.crt")

	tests := map[string]struct {
		url     string
		environ map[string]string
		want    string
	}{
		"a CA file": {
			url: url + "&sslrootcert=" + certFile,
		},
		"a CA file and a client certificate": {
			url: url + "&sslrootcert=" + certFile + "&sslcert=" + certFile + "&sslkey=" + keyFile,
		},
		"a CA file in PGSSLROOTCERT": {
			url:     url,
			environ: map[string]string{"PGSSLROOTCERT": certFile},
		},
		"a missing sslrootcert": {
			url:  url + "&sslrootcert=" + missing,
			want: "can't be used, because sslrootcert or PGSSLROOTCERT names a file that can't be read: no such file or directory",
		},
		"a missing PGSSLROOTCERT": {
			url:     url,
			environ: map[string]string{"PGSSLROOTCERT": missing},
			want:    "can't be used, because sslrootcert or PGSSLROOTCERT names a file that can't be read: no such file or directory",
		},
		// Unlike libpq, pgx reads sslrootcert even if it doesn't use TLS.
		"a missing sslrootcert with sslmode=disable": {
			url:  baseURL + "?sslmode=disable&sslrootcert=" + missing,
			want: "can't be used, because sslrootcert or PGSSLROOTCERT names a file that can't be read: no such file or directory",
		},
		"a missing sslcert": {
			url:  url + "&sslcert=" + missing + "&sslkey=" + keyFile,
			want: "can't be used, because sslcert or PGSSLCERT names a file that can't be read: no such file or directory",
		},
		"a missing PGSSLKEY": {
			url:     url,
			environ: map[string]string{"PGSSLCERT": certFile, "PGSSLKEY": missing},
			want:    "can't be used, because sslkey or PGSSLKEY names a file that can't be read: no such file or directory",
		},
		// pgx takes the password as part of the path here, because the space
		// before it is missing.
		"a missing sslrootcert that runs into the password": {
			url:  "host=" + host + " user=phemera sslmode=verify-full sslrootcert=" + missing + "password=" + password,
			want: "can't be used, because sslrootcert or PGSSLROOTCERT names a file that can't be read: no such file or directory",
		},
		// pgx can't find the password to mask in this URL, because the @ is
		// missing.
		"an invalid URL": {
			url:  "postgres://phemera:" + password + "host:x/phemera",
			want: invalidURL,
		},
		"an invalid PG variable": {
			url:     baseURL,
			environ: map[string]string{"PGSSLMODE": "bogus"},
			want:    invalidURL,
		},
		"a CA file that isn't a certificate": {
			url:  url + "&sslrootcert=" + keyFile,
			want: invalidURL,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// arrange
			for variable, value := range test.environ {
				t.Setenv(variable, value)
			}

			// act
			config, err := dburl.Parse(test.url)

			// assert
			if test.want == "" {
				require.NoError(t, err)
				assert.NotNil(t, config)

				return
			}

			require.EqualError(t, err, test.want)
			assert.NotContains(t, err.Error(), password)
			assert.NotContains(t, err.Error(), host)
		})
	}
}

func TestParseNamesATLSFileThatItMayNotRead(t *testing.T) {
	t.Parallel()

	if os.Geteuid() == 0 {
		t.Skip("root reads the file whatever its mode")
	}

	// arrange
	certFile, _ := writeKeyPair(t)
	require.NoError(t, os.Chmod(certFile, 0o000))

	// act
	_, err := dburl.Parse(url + "&sslrootcert=" + certFile)

	// assert
	assert.EqualError(t, err,
		"can't be used, because sslrootcert or PGSSLROOTCERT names a file that can't be read: permission denied")
}

// writeKeyPair writes a self-signed certificate, which also serves as a CA
// file.
func writeKeyPair(t testing.TB) (certFile, keyFile string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{SerialNumber: big.NewInt(1), IsCA: true, BasicConstraintsValid: true}
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
