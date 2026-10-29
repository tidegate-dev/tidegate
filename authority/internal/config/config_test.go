package config_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/authority/internal/config"
	"github.com/tidegate-dev/tidegate/internal/logging"
)

func TestDefaultsApplyWithOnlyTheDatabaseURL(t *testing.T) {
	t.Parallel()

	// act
	cfg, err := config.Load(withDatabaseURL())

	// assert
	require.NoError(t, err)
	assert.Equal(t, defaultConfig(), cfg)
}

func TestVariablesOverrideDefaults(t *testing.T) {
	t.Parallel()

	// arrange
	environ := withDatabaseURL(
		"TIDEGATE_AUTHORITY__SERVER__LISTEN_ADDRESS=:9090",
		"TIDEGATE_AUTHORITY__METRICS__LISTEN_ADDRESS=:9100",
		"TIDEGATE_AUTHORITY__LOG__LEVEL=debug",
		"TIDEGATE_AUTHORITY__LOG__FORMAT=text",
		"TIDEGATE_AUTHORITY__DATABASE__MIGRATE_ON_START=true",
	)

	// act
	cfg, err := config.Load(environ)

	// assert
	require.NoError(t, err)
	assert.Equal(t, config.Config{
		Server:   config.Server{ListenAddress: ":9090"},
		Metrics:  config.Metrics{ListenAddress: ":9100"},
		Database: config.Database{URL: databaseURL, MigrateOnStart: true},
		Log:      config.Log{Level: slog.LevelDebug, Format: logging.FormatText},
	}, cfg)
}

func TestVariablesThatAreNotSettingsAreIgnored(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"without prefix":                    "LOG__LEVEL=loud",
		"without the AUTHORITY part":        "TIDEGATE_LOG__LEVEL=loud",
		"kubernetes service host":           "TIDEGATE_AUTHORITY_SERVICE_HOST=10.0.0.1",
		"kubernetes service port":           "TIDEGATE_AUTHORITY_PORT=tcp://10.0.0.1:8080",
		"single underscore after AUTHORITY": "TIDEGATE_AUTHORITY_LOG__LEVEL=loud",
	}

	for name, variable := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			cfg, err := config.Load(withDatabaseURL(variable))

			// assert
			require.NoError(t, err)
			assert.Equal(t, defaultConfig(), cfg)
		})
	}
}

func TestEmptyVariableFallsBackToDefault(t *testing.T) {
	t.Parallel()

	// act
	cfg, err := config.Load(withDatabaseURL("TIDEGATE_AUTHORITY__LOG__LEVEL="))

	// assert
	require.NoError(t, err)
	assert.Equal(t, defaultConfig(), cfg)
}

func TestNamesAndValuesIgnoreLetterCase(t *testing.T) {
	t.Parallel()

	// arrange
	environ := withDatabaseURL(
		"TIDEGATE_AUTHORITY__Log__Level=DEBUG",
		"TIDEGATE_AUTHORITY__LOG__FORMAT=Text",
	)

	// act
	cfg, err := config.Load(environ)

	// assert
	require.NoError(t, err)
	assert.Equal(t, slog.LevelDebug, cfg.Log.Level)
	assert.Equal(t, logging.FormatText, cfg.Log.Format)
}

func TestValueMayContainEqualsSign(t *testing.T) {
	t.Parallel()

	// act
	cfg, err := config.Load(withDatabaseURL("TIDEGATE_AUTHORITY__SERVER__LISTEN_ADDRESS=a=b:80"))

	// assert
	require.NoError(t, err)
	assert.Equal(t, "a=b:80", cfg.Server.ListenAddress)
}

func TestEveryProblemIsReported(t *testing.T) {
	t.Parallel()

	// arrange
	environ := withDatabaseURL(
		"TIDEGATE_AUTHORITY__LOG__LEVEL=loud",
		"TIDEGATE_AUTHORITY__LOG__LEVL=debug",
		"TIDEGATE_AUTHORITY__LOG__FORMAT=xml",
		"TIDEGATE_AUTHORITY__SERVER__PORT=80",
	)

	// act
	_, err := config.Load(environ)

	// assert
	assert.EqualError(t, err, strings.Join([]string{
		"TIDEGATE_AUTHORITY__LOG__LEVL: unknown setting",
		"TIDEGATE_AUTHORITY__SERVER__PORT: unknown setting",
		`TIDEGATE_AUTHORITY__LOG__LEVEL: unknown log level "loud", want debug, info, warn or error`,
		`TIDEGATE_AUTHORITY__LOG__FORMAT: unknown log format "xml", want json or text`,
	}, "\n"))
}

func TestInvalidValuesNameTheirVariable(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		variable string
		want     string
	}{
		"log level":      {variable: "TIDEGATE_AUTHORITY__LOG__LEVEL=loud", want: `TIDEGATE_AUTHORITY__LOG__LEVEL: unknown log level "loud", want debug, info, warn or error`},
		"log format":     {variable: "TIDEGATE_AUTHORITY__LOG__FORMAT=xml", want: `TIDEGATE_AUTHORITY__LOG__FORMAT: unknown log format "xml", want json or text`},
		"listen address": {variable: "TIDEGATE_AUTHORITY__SERVER__LISTEN_ADDRESS=8080", want: `TIDEGATE_AUTHORITY__SERVER__LISTEN_ADDRESS: invalid listen address "8080", want host:port`},
		"port":           {variable: "TIDEGATE_AUTHORITY__SERVER__LISTEN_ADDRESS=:99999", want: `TIDEGATE_AUTHORITY__SERVER__LISTEN_ADDRESS: invalid port in listen address ":99999"`},
		"unknown":        {variable: "TIDEGATE_AUTHORITY__LOG__LEVL=debug", want: "TIDEGATE_AUTHORITY__LOG__LEVL: unknown setting"},
		"too deep":       {variable: "TIDEGATE_AUTHORITY__LOG__LEVEL__X=debug", want: "TIDEGATE_AUTHORITY__LOG__LEVEL__X: unknown setting"},
		"single level":   {variable: "TIDEGATE_AUTHORITY__LOG_LEVEL=debug", want: "TIDEGATE_AUTHORITY__LOG_LEVEL: unknown setting"},
		"prefix only":    {variable: "TIDEGATE_AUTHORITY__=debug", want: "TIDEGATE_AUTHORITY__: unknown setting"},
		"migrate":        {variable: "TIDEGATE_AUTHORITY__DATABASE__MIGRATE_ON_START=yes", want: `TIDEGATE_AUTHORITY__DATABASE__MIGRATE_ON_START: invalid value "yes", want true or false`},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			_, err := config.Load(withDatabaseURL(tt.variable))

			// assert
			assert.EqualError(t, err, tt.want)
		})
	}
}

func TestCertificateAndKeyTogetherEnableTLS(t *testing.T) {
	t.Parallel()

	// arrange
	certFile, keyFile := writeKeyPair(t)

	// act
	cfg, err := config.Load(withDatabaseURL(
		"TIDEGATE_AUTHORITY__SERVER__TLS__CERT_FILE="+certFile,
		"TIDEGATE_AUTHORITY__SERVER__TLS__KEY_FILE="+keyFile,
	))

	// assert
	require.NoError(t, err)
	assert.NotNil(t, cfg.Server.Certificate)
}

func TestCertificateOrKeyAloneIsRejected(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		variable string
		want     string
	}{
		"certificate only": {
			variable: "TIDEGATE_AUTHORITY__SERVER__TLS__CERT_FILE=cert.pem",
			want:     "TIDEGATE_AUTHORITY__SERVER__TLS__KEY_FILE: must be set together with TIDEGATE_AUTHORITY__SERVER__TLS__CERT_FILE",
		},
		"key only": {
			variable: "TIDEGATE_AUTHORITY__SERVER__TLS__KEY_FILE=key.pem",
			want:     "TIDEGATE_AUTHORITY__SERVER__TLS__CERT_FILE: must be set together with TIDEGATE_AUTHORITY__SERVER__TLS__KEY_FILE",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			_, err := config.Load(withDatabaseURL(tt.variable))

			// assert
			assert.EqualError(t, err, tt.want)
		})
	}
}

func TestMissingKeyPairNamesTheCertificateVariable(t *testing.T) {
	t.Parallel()

	// arrange
	directory := t.TempDir()

	// act
	_, err := config.Load(withDatabaseURL(
		"TIDEGATE_AUTHORITY__SERVER__TLS__CERT_FILE="+filepath.Join(directory, "cert.pem"),
		"TIDEGATE_AUTHORITY__SERVER__TLS__KEY_FILE="+filepath.Join(directory, "key.pem"),
	))

	// assert
	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "TIDEGATE_AUTHORITY__SERVER__TLS__CERT_FILE: loading the key pair: "), err.Error())
}

func TestDatabaseURLIsRequired(t *testing.T) {
	t.Parallel()

	// act
	_, err := config.Load(nil)

	// assert
	assert.EqualError(t, err, "TIDEGATE_AUTHORITY__DATABASE__URL: must be set")
}

func TestInvalidDatabaseURLIsRejectedWithoutTheURL(t *testing.T) {
	t.Parallel()

	// arrange
	// pgx can't find the password to mask in this URL, because the @ is missing.
	environ := []string{"TIDEGATE_AUTHORITY__DATABASE__URL=postgres://tidegate:hunter2host:x/tidegate"}

	// act
	_, err := config.Load(environ)

	// assert
	assert.EqualError(t, err, "TIDEGATE_AUTHORITY__DATABASE__URL: "+invalidDatabaseURL)
}

func TestInvalidPGVariableIsReportedWithTheDatabaseURL(t *testing.T) {
	// arrange
	t.Setenv("PGSSLMODE", "bogus")

	// act
	_, err := config.Load(withDatabaseURL())

	// assert
	assert.EqualError(t, err, "TIDEGATE_AUTHORITY__DATABASE__URL: "+invalidDatabaseURL)
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
	certFile = filepath.Join(directory, "cert.pem")
	keyFile = filepath.Join(directory, "key.pem")

	require.NoError(t, os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))
	require.NoError(t, os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0o600))

	return certFile, keyFile
}

const (
	databaseURL        = "postgres://localhost/tidegate"
	invalidDatabaseURL = "isn't a valid Postgres connection URL, or a PG variable like PGSSLMODE is invalid"
)

func withDatabaseURL(variables ...string) []string {
	return append([]string{"TIDEGATE_AUTHORITY__DATABASE__URL=" + databaseURL}, variables...)
}

func defaultConfig() config.Config {
	return config.Config{
		Server:   config.Server{ListenAddress: "127.0.0.1:8080"},
		Metrics:  config.Metrics{ListenAddress: "127.0.0.1:9464"},
		Database: config.Database{URL: databaseURL},
		Log:      config.Log{Level: slog.LevelInfo, Format: logging.FormatJSON},
	}
}
