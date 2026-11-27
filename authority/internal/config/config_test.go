package config_test

import (
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phemera-dev/phemera/authority/internal/config"
	"github.com/phemera-dev/phemera/internal/logging"
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
		"PHEMERA_AUTHORITY__EXTERNAL_API__LISTEN_ADDRESS=:9090",
		"PHEMERA_AUTHORITY__INTERNAL_API__LISTEN_ADDRESS=:9091",
		"PHEMERA_AUTHORITY__METRICS__LISTEN_ADDRESS=:9100",
		"PHEMERA_AUTHORITY__LOG__LEVEL=debug",
		"PHEMERA_AUTHORITY__LOG__FORMAT=text",
		"PHEMERA_AUTHORITY__DATABASE__MIGRATE_ON_START=true",
		"PHEMERA_AUTHORITY__LOGIN__TOKEN_LIFETIME=90m",
	)

	// act
	cfg, err := config.Load(environ)

	// assert
	require.NoError(t, err)
	assert.Equal(t, config.Config{
		ExternalAPI: config.ExternalAPI{ListenAddress: ":9090", TLS: defaultConfig().ExternalAPI.TLS},
		InternalAPI: config.InternalAPI{ListenAddress: ":9091", TLS: defaultConfig().InternalAPI.TLS},
		Metrics:     config.Metrics{ListenAddress: ":9100"},
		Database:    config.Database{URL: databaseURL, MigrateOnStart: true},
		Login:       config.Login{TokenLifetime: 90 * time.Minute},
		Log:         config.Log{Level: slog.LevelDebug, Format: logging.FormatText},
	}, cfg)
}

func TestVariablesThatAreNotSettingsAreIgnored(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"without prefix":                    "LOG__LEVEL=loud",
		"without the AUTHORITY part":        "PHEMERA_LOG__LEVEL=loud",
		"kubernetes service host":           "PHEMERA_AUTHORITY_SERVICE_HOST=10.0.0.1",
		"kubernetes service port":           "PHEMERA_AUTHORITY_PORT=tcp://10.0.0.1:8080",
		"single underscore after AUTHORITY": "PHEMERA_AUTHORITY_LOG__LEVEL=loud",
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
	cfg, err := config.Load(withDatabaseURL("PHEMERA_AUTHORITY__LOG__LEVEL="))

	// assert
	require.NoError(t, err)
	assert.Equal(t, defaultConfig(), cfg)
}

func TestNamesAndValuesIgnoreLetterCase(t *testing.T) {
	t.Parallel()

	// arrange
	environ := withDatabaseURL(
		"PHEMERA_AUTHORITY__Log__Level=DEBUG",
		"PHEMERA_AUTHORITY__LOG__FORMAT=Text",
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
	cfg, err := config.Load(withDatabaseURL("PHEMERA_AUTHORITY__EXTERNAL_API__LISTEN_ADDRESS=a=b:80"))

	// assert
	require.NoError(t, err)
	assert.Equal(t, "a=b:80", cfg.ExternalAPI.ListenAddress)
}

func TestEveryProblemIsReported(t *testing.T) {
	t.Parallel()

	// arrange
	environ := withDatabaseURL(
		"PHEMERA_AUTHORITY__LOG__LEVEL=loud",
		"PHEMERA_AUTHORITY__LOG__LEVL=debug",
		"PHEMERA_AUTHORITY__LOG__FORMAT=xml",
		"PHEMERA_AUTHORITY__EXTERNAL_API__PORT=80",
	)

	// act
	_, err := config.Load(environ)

	// assert
	assert.EqualError(t, err, strings.Join([]string{
		"PHEMERA_AUTHORITY__LOG__LEVL: unknown setting",
		"PHEMERA_AUTHORITY__EXTERNAL_API__PORT: unknown setting",
		`PHEMERA_AUTHORITY__LOG__LEVEL: unknown log level "loud", want debug, info, warn or error`,
		`PHEMERA_AUTHORITY__LOG__FORMAT: unknown log format "xml", want json or text`,
	}, "\n"))
}

func TestInvalidValuesNameTheirVariable(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		variable string
		want     string
	}{
		"log level":               {variable: "PHEMERA_AUTHORITY__LOG__LEVEL=loud", want: `PHEMERA_AUTHORITY__LOG__LEVEL: unknown log level "loud", want debug, info, warn or error`},
		"log format":              {variable: "PHEMERA_AUTHORITY__LOG__FORMAT=xml", want: `PHEMERA_AUTHORITY__LOG__FORMAT: unknown log format "xml", want json or text`},
		"external listen address": {variable: "PHEMERA_AUTHORITY__EXTERNAL_API__LISTEN_ADDRESS=8080", want: `PHEMERA_AUTHORITY__EXTERNAL_API__LISTEN_ADDRESS: invalid listen address "8080", want host:port`},
		"external port":           {variable: "PHEMERA_AUTHORITY__EXTERNAL_API__LISTEN_ADDRESS=:99999", want: `PHEMERA_AUTHORITY__EXTERNAL_API__LISTEN_ADDRESS: invalid port in listen address ":99999"`},
		"internal listen address": {variable: "PHEMERA_AUTHORITY__INTERNAL_API__LISTEN_ADDRESS=8081", want: `PHEMERA_AUTHORITY__INTERNAL_API__LISTEN_ADDRESS: invalid listen address "8081", want host:port`},
		"unknown":                 {variable: "PHEMERA_AUTHORITY__LOG__LEVL=debug", want: "PHEMERA_AUTHORITY__LOG__LEVL: unknown setting"},
		"too deep":                {variable: "PHEMERA_AUTHORITY__LOG__LEVEL__X=debug", want: "PHEMERA_AUTHORITY__LOG__LEVEL__X: unknown setting"},
		"single level":            {variable: "PHEMERA_AUTHORITY__LOG_LEVEL=debug", want: "PHEMERA_AUTHORITY__LOG_LEVEL: unknown setting"},
		"prefix only":             {variable: "PHEMERA_AUTHORITY__=debug", want: "PHEMERA_AUTHORITY__: unknown setting"},
		"migrate":                 {variable: "PHEMERA_AUTHORITY__DATABASE__MIGRATE_ON_START=yes", want: `PHEMERA_AUTHORITY__DATABASE__MIGRATE_ON_START: invalid value "yes", want true or false`},
		"token lifetime":          {variable: "PHEMERA_AUTHORITY__LOGIN__TOKEN_LIFETIME=12", want: `PHEMERA_AUTHORITY__LOGIN__TOKEN_LIFETIME: invalid duration "12", want a number with a unit, like 12h or 30m`},
		"short lifetime":          {variable: "PHEMERA_AUTHORITY__LOGIN__TOKEN_LIFETIME=999ms", want: `PHEMERA_AUTHORITY__LOGIN__TOKEN_LIFETIME: must be at least 1s`},
		"negative":                {variable: "PHEMERA_AUTHORITY__LOGIN__TOKEN_LIFETIME=-1h", want: `PHEMERA_AUTHORITY__LOGIN__TOKEN_LIFETIME: must be at least 1s`},
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

func TestOverlappingListenAddressesAreRejected(t *testing.T) {
	t.Parallel()

	const (
		external = "PHEMERA_AUTHORITY__EXTERNAL_API__LISTEN_ADDRESS"
		internal = "PHEMERA_AUTHORITY__INTERNAL_API__LISTEN_ADDRESS"
		metrics  = "PHEMERA_AUTHORITY__METRICS__LISTEN_ADDRESS"
	)

	tests := map[string]struct {
		variables []string
		want      string
	}{
		"same address": {
			variables: []string{external + "=127.0.0.1:9000", internal + "=127.0.0.1:9000"},
			want:      internal + `: "127.0.0.1:9000" overlaps with "127.0.0.1:9000" in ` + external,
		},
		"every interface and a default": {
			variables: []string{external + "=:8081"},
			want:      internal + `: "127.0.0.1:8081" overlaps with ":8081" in ` + external,
		},
		"every IPv4 interface": {
			variables: []string{external + "=10.0.0.1:9000", internal + "=0.0.0.0:9000"},
			want:      internal + `: "0.0.0.0:9000" overlaps with "10.0.0.1:9000" in ` + external,
		},
		"every IPv6 interface": {
			variables: []string{external + "=[::]:9000", internal + "=127.0.0.1:9000"},
			want:      internal + `: "127.0.0.1:9000" overlaps with "[::]:9000" in ` + external,
		},
		"one IP written in two ways": {
			variables: []string{external + "=[::1]:9000", internal + "=[0:0:0:0:0:0:0:1]:9000"},
			want:      internal + `: "[0:0:0:0:0:0:0:1]:9000" overlaps with "[::1]:9000" in ` + external,
		},
		"one name in two cases": {
			variables: []string{external + "=localhost:9000", internal + "=LOCALHOST:9000"},
			want:      internal + `: "LOCALHOST:9000" overlaps with "localhost:9000" in ` + external,
		},
		"metrics and an API": {
			variables: []string{metrics + "=:8080"},
			want:      metrics + `: ":8080" overlaps with "127.0.0.1:8080" in ` + external,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			_, err := config.Load(withDatabaseURL(tt.variables...))

			// assert
			assert.EqualError(t, err, tt.want)
		})
	}
}

func TestListenAddressesThatCanBothBeBoundAreAccepted(t *testing.T) {
	t.Parallel()

	const (
		external = "PHEMERA_AUTHORITY__EXTERNAL_API__LISTEN_ADDRESS"
		internal = "PHEMERA_AUTHORITY__INTERNAL_API__LISTEN_ADDRESS"
	)

	tests := map[string][]string{
		"one port on two IPs":          {external + "=127.0.0.1:9000", internal + "=127.0.0.2:9000"},
		"two ports on every interface": {external + "=:9000", internal + "=:9001"},
		"a port the system picks":      {external + "=127.0.0.1:0", internal + "=127.0.0.1:0"},
		// A name can resolve to any address, so only the bind can tell.
		"a name and an IP": {external + "=localhost:9000", internal + "=127.0.0.1:9000"},
	}

	for name, variables := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			_, err := config.Load(withDatabaseURL(variables...))

			// assert
			assert.NoError(t, err)
		})
	}
}

func TestExternalTLSFilesCanBeSet(t *testing.T) {
	t.Parallel()

	// act
	cfg, err := config.Load(withDatabaseURL(
		"PHEMERA_AUTHORITY__EXTERNAL_API__TLS__CERT_FILE=/srv/tls/cert.pem",
		"PHEMERA_AUTHORITY__EXTERNAL_API__TLS__KEY_FILE=/srv/tls/key.pem",
	))

	// assert
	require.NoError(t, err)
	assert.Equal(t, config.ExternalTLS{CertFile: "/srv/tls/cert.pem", KeyFile: "/srv/tls/key.pem"}, cfg.ExternalAPI.TLS)
}

func TestExternalTLSCanBeDisabled(t *testing.T) {
	t.Parallel()

	// act
	cfg, err := config.Load(withDatabaseURL("PHEMERA_AUTHORITY__EXTERNAL_API__TLS__DISABLED=true"))

	// assert
	require.NoError(t, err)
	assert.True(t, cfg.ExternalAPI.TLS.Disabled)
}

func TestDisabledExternalTLSRejectsItsFiles(t *testing.T) {
	t.Parallel()

	for name, variable := range map[string]string{
		"certificate":                     "PHEMERA_AUTHORITY__EXTERNAL_API__TLS__CERT_FILE=/srv/tls/cert.pem",
		"key":                             "PHEMERA_AUTHORITY__EXTERNAL_API__TLS__KEY_FILE=/srv/tls/key.pem",
		"certificate at its default path": "PHEMERA_AUTHORITY__EXTERNAL_API__TLS__CERT_FILE=/etc/phemera/authority/external-api-tls/tls.crt",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			_, err := config.Load(withDatabaseURL("PHEMERA_AUTHORITY__EXTERNAL_API__TLS__DISABLED=true", variable))

			// assert
			assert.EqualError(t, err, "PHEMERA_AUTHORITY__EXTERNAL_API__TLS__DISABLED: "+
				"can't be true while PHEMERA_AUTHORITY__EXTERNAL_API__TLS__CERT_FILE or PHEMERA_AUTHORITY__EXTERNAL_API__TLS__KEY_FILE is set")
		})
	}
}

func TestDisabledExternalTLSAllowsInternalTLSFiles(t *testing.T) {
	t.Parallel()

	// act
	_, err := config.Load(withDatabaseURL(
		"PHEMERA_AUTHORITY__EXTERNAL_API__TLS__DISABLED=true",
		"PHEMERA_AUTHORITY__INTERNAL_API__TLS__CERT_FILE=/srv/tls/cert.pem",
		"PHEMERA_AUTHORITY__INTERNAL_API__TLS__KEY_FILE=/srv/tls/key.pem",
	))

	// assert
	assert.NoError(t, err)
}

func TestExternalTLSDisabledVariableNamesTheSettingThatLoadReads(t *testing.T) {
	t.Parallel()

	// act
	cfg, err := config.Load(withDatabaseURL(config.ExternalTLSDisabledVariable() + "=true"))

	// assert
	require.NoError(t, err)
	assert.True(t, cfg.ExternalAPI.TLS.Disabled)
}

func TestInternalTLSFilesCanBeSet(t *testing.T) {
	t.Parallel()

	// act
	cfg, err := config.Load(withDatabaseURL(
		"PHEMERA_AUTHORITY__INTERNAL_API__TLS__CERT_FILE=/srv/tls/cert.pem",
		"PHEMERA_AUTHORITY__INTERNAL_API__TLS__KEY_FILE=/srv/tls/key.pem",
	))

	// assert
	require.NoError(t, err)
	assert.Equal(t, config.InternalTLS{CertFile: "/srv/tls/cert.pem", KeyFile: "/srv/tls/key.pem"}, cfg.InternalAPI.TLS)
}

func TestInternalTLSCantBeDisabled(t *testing.T) {
	t.Parallel()

	// act
	_, err := config.Load(withDatabaseURL("PHEMERA_AUTHORITY__INTERNAL_API__TLS__DISABLED=true"))

	// assert
	assert.EqualError(t, err, "PHEMERA_AUTHORITY__INTERNAL_API__TLS__DISABLED: unknown setting")
}

func TestDatabaseURLIsRequired(t *testing.T) {
	t.Parallel()

	// act
	_, err := config.Load(nil)

	// assert
	assert.EqualError(t, err, "PHEMERA_AUTHORITY__DATABASE__URL: must be set")
}

func TestInvalidDatabaseURLIsRejectedWithoutTheURL(t *testing.T) {
	t.Parallel()

	// arrange
	// pgx can't find the password to mask in this URL, because the @ is missing.
	environ := []string{"PHEMERA_AUTHORITY__DATABASE__URL=postgres://phemera:hunter2host:x/phemera"}

	// act
	_, err := config.Load(environ)

	// assert
	assert.EqualError(t, err, "PHEMERA_AUTHORITY__DATABASE__URL: "+invalidDatabaseURL)
}

func TestInvalidPGVariableIsReportedWithTheDatabaseURL(t *testing.T) {
	// arrange
	t.Setenv("PGSSLMODE", "bogus")

	// act
	_, err := config.Load(withDatabaseURL())

	// assert
	assert.EqualError(t, err, "PHEMERA_AUTHORITY__DATABASE__URL: "+invalidDatabaseURL)
}

func TestMissingTLSFileOfTheDatabaseURLIsNamed(t *testing.T) {
	t.Parallel()

	// arrange
	missing := filepath.Join(t.TempDir(), "ca.crt")
	environ := []string{"PHEMERA_AUTHORITY__DATABASE__URL=" + databaseURL + "?sslmode=verify-full&sslrootcert=" + missing}

	// act
	_, err := config.Load(environ)

	// assert
	assert.EqualError(t, err, "PHEMERA_AUTHORITY__DATABASE__URL: can't be used, "+
		"because sslrootcert or PGSSLROOTCERT names a file that can't be read: no such file or directory")
}

const (
	databaseURL        = "postgres://localhost/phemera"
	invalidDatabaseURL = "isn't a valid Postgres connection URL, or a PG variable like PGSSLMODE is invalid"
)

func withDatabaseURL(variables ...string) []string {
	return append([]string{"PHEMERA_AUTHORITY__DATABASE__URL=" + databaseURL}, variables...)
}

func defaultConfig() config.Config {
	return config.Config{
		ExternalAPI: config.ExternalAPI{
			ListenAddress: "127.0.0.1:8080",
			TLS:           config.ExternalTLS{CertFile: "/etc/phemera/authority/external-api-tls/tls.crt", KeyFile: "/etc/phemera/authority/external-api-tls/tls.key"},
		},
		InternalAPI: config.InternalAPI{
			ListenAddress: "127.0.0.1:8081",
			TLS: config.InternalTLS{
				CertFile: "/etc/phemera/authority/internal-api-tls/tls.crt",
				KeyFile:  "/etc/phemera/authority/internal-api-tls/tls.key",
			},
		},
		Metrics:  config.Metrics{ListenAddress: "127.0.0.1:9464"},
		Database: config.Database{URL: databaseURL},
		Login:    config.Login{TokenLifetime: 12 * time.Hour},
		Log:      config.Log{Level: slog.LevelInfo, Format: logging.FormatJSON},
	}
}
