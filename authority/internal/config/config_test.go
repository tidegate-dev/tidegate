package config_test

import (
	"log/slog"
	"strings"
	"testing"
	"time"

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
		"TIDEGATE_AUTHORITY__LOGIN__TOKEN_LIFETIME=90m",
	)

	// act
	cfg, err := config.Load(environ)

	// assert
	require.NoError(t, err)
	assert.Equal(t, config.Config{
		Server:   config.Server{ListenAddress: ":9090", TLS: defaultConfig().Server.TLS},
		Metrics:  config.Metrics{ListenAddress: ":9100"},
		Database: config.Database{URL: databaseURL, MigrateOnStart: true},
		Login:    config.Login{TokenLifetime: 90 * time.Minute},
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
		"token lifetime": {variable: "TIDEGATE_AUTHORITY__LOGIN__TOKEN_LIFETIME=12", want: `TIDEGATE_AUTHORITY__LOGIN__TOKEN_LIFETIME: invalid duration "12", want a number with a unit, like 12h or 30m`},
		"short lifetime": {variable: "TIDEGATE_AUTHORITY__LOGIN__TOKEN_LIFETIME=999ms", want: `TIDEGATE_AUTHORITY__LOGIN__TOKEN_LIFETIME: must be at least 1s`},
		"negative":       {variable: "TIDEGATE_AUTHORITY__LOGIN__TOKEN_LIFETIME=-1h", want: `TIDEGATE_AUTHORITY__LOGIN__TOKEN_LIFETIME: must be at least 1s`},
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

func TestTLSFilesCanBeSet(t *testing.T) {
	t.Parallel()

	// act
	cfg, err := config.Load(withDatabaseURL(
		"TIDEGATE_AUTHORITY__SERVER__TLS__CERT_FILE=/srv/tls/cert.pem",
		"TIDEGATE_AUTHORITY__SERVER__TLS__KEY_FILE=/srv/tls/key.pem",
	))

	// assert
	require.NoError(t, err)
	assert.Equal(t, config.TLS{CertFile: "/srv/tls/cert.pem", KeyFile: "/srv/tls/key.pem"}, cfg.Server.TLS)
}

func TestTLSCanBeDisabled(t *testing.T) {
	t.Parallel()

	// act
	cfg, err := config.Load(withDatabaseURL("TIDEGATE_AUTHORITY__SERVER__TLS__DISABLED=true"))

	// assert
	require.NoError(t, err)
	assert.True(t, cfg.Server.TLS.Disabled)
}

func TestDisabledTLSRejectsTLSFiles(t *testing.T) {
	t.Parallel()

	for name, variable := range map[string]string{
		"certificate":                     "TIDEGATE_AUTHORITY__SERVER__TLS__CERT_FILE=/srv/tls/cert.pem",
		"key":                             "TIDEGATE_AUTHORITY__SERVER__TLS__KEY_FILE=/srv/tls/key.pem",
		"certificate at its default path": "TIDEGATE_AUTHORITY__SERVER__TLS__CERT_FILE=/etc/tidegate/authority/tls/tls.crt",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			_, err := config.Load(withDatabaseURL("TIDEGATE_AUTHORITY__SERVER__TLS__DISABLED=true", variable))

			// assert
			assert.EqualError(t, err, "TIDEGATE_AUTHORITY__SERVER__TLS__DISABLED: "+
				"can't be true while TIDEGATE_AUTHORITY__SERVER__TLS__CERT_FILE or TIDEGATE_AUTHORITY__SERVER__TLS__KEY_FILE is set")
		})
	}
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

const (
	databaseURL        = "postgres://localhost/tidegate"
	invalidDatabaseURL = "isn't a valid Postgres connection URL, or a PG variable like PGSSLMODE is invalid"
)

func withDatabaseURL(variables ...string) []string {
	return append([]string{"TIDEGATE_AUTHORITY__DATABASE__URL=" + databaseURL}, variables...)
}

func defaultConfig() config.Config {
	return config.Config{
		Server: config.Server{
			ListenAddress: "127.0.0.1:8080",
			TLS:           config.TLS{CertFile: "/etc/tidegate/authority/tls/tls.crt", KeyFile: "/etc/tidegate/authority/tls/tls.key"},
		},
		Metrics:  config.Metrics{ListenAddress: "127.0.0.1:9464"},
		Database: config.Database{URL: databaseURL},
		Login:    config.Login{TokenLifetime: 12 * time.Hour},
		Log:      config.Log{Level: slog.LevelInfo, Format: logging.FormatJSON},
	}
}
