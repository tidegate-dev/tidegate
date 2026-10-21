package config_test

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/control-plane/internal/config"
	"github.com/tidegate-dev/tidegate/internal/logging"
)

func TestDefaultsApplyWithoutVariables(t *testing.T) {
	t.Parallel()

	// act
	cfg, err := config.Load(nil)

	// assert
	require.NoError(t, err)
	assert.Equal(t, defaultConfig(), cfg)
}

func TestVariablesOverrideDefaults(t *testing.T) {
	t.Parallel()

	// arrange
	environ := []string{
		"TIDEGATE_CP__SERVER__LISTEN_ADDRESS=:9090",
		"TIDEGATE_CP__METRICS__LISTEN_ADDRESS=:9100",
		"TIDEGATE_CP__LOG__LEVEL=debug",
		"TIDEGATE_CP__LOG__FORMAT=text",
	}

	// act
	cfg, err := config.Load(environ)

	// assert
	require.NoError(t, err)
	assert.Equal(t, config.Config{
		Server:  config.Server{ListenAddress: ":9090"},
		Metrics: config.Metrics{ListenAddress: ":9100"},
		Log:     config.Log{Level: slog.LevelDebug, Format: logging.FormatText},
	}, cfg)
}

func TestVariablesThatAreNotSettingsAreIgnored(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"without prefix":             "LOG__LEVEL=loud",
		"without the CP part":        "TIDEGATE_LOG__LEVEL=loud",
		"kubernetes service host":    "TIDEGATE_CP_SERVICE_HOST=10.0.0.1",
		"kubernetes service port":    "TIDEGATE_CP_PORT=tcp://10.0.0.1:8080",
		"single underscore after CP": "TIDEGATE_CP_LOG__LEVEL=loud",
	}

	for name, variable := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			cfg, err := config.Load([]string{variable})

			// assert
			require.NoError(t, err)
			assert.Equal(t, defaultConfig(), cfg)
		})
	}
}

func TestEmptyVariableFallsBackToDefault(t *testing.T) {
	t.Parallel()

	// act
	cfg, err := config.Load([]string{"TIDEGATE_CP__LOG__LEVEL="})

	// assert
	require.NoError(t, err)
	assert.Equal(t, defaultConfig(), cfg)
}

func TestNamesAndValuesIgnoreLetterCase(t *testing.T) {
	t.Parallel()

	// arrange
	environ := []string{
		"TIDEGATE_CP__Log__Level=DEBUG",
		"TIDEGATE_CP__LOG__FORMAT=Text",
	}

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
	cfg, err := config.Load([]string{"TIDEGATE_CP__SERVER__LISTEN_ADDRESS=a=b:80"})

	// assert
	require.NoError(t, err)
	assert.Equal(t, "a=b:80", cfg.Server.ListenAddress)
}

func TestEveryProblemIsReported(t *testing.T) {
	t.Parallel()

	// arrange
	environ := []string{
		"TIDEGATE_CP__LOG__LEVEL=loud",
		"TIDEGATE_CP__LOG__LEVL=debug",
		"TIDEGATE_CP__LOG__FORMAT=xml",
		"TIDEGATE_CP__SERVER__PORT=80",
	}

	// act
	_, err := config.Load(environ)

	// assert
	assert.EqualError(t, err, strings.Join([]string{
		"TIDEGATE_CP__LOG__LEVL: unknown setting",
		"TIDEGATE_CP__SERVER__PORT: unknown setting",
		`TIDEGATE_CP__LOG__LEVEL: unknown log level "loud", want debug, info, warn or error`,
		`TIDEGATE_CP__LOG__FORMAT: unknown log format "xml", want json or text`,
	}, "\n"))
}

func TestInvalidValuesNameTheirVariable(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		variable string
		want     string
	}{
		"log level":      {variable: "TIDEGATE_CP__LOG__LEVEL=loud", want: `TIDEGATE_CP__LOG__LEVEL: unknown log level "loud", want debug, info, warn or error`},
		"log format":     {variable: "TIDEGATE_CP__LOG__FORMAT=xml", want: `TIDEGATE_CP__LOG__FORMAT: unknown log format "xml", want json or text`},
		"listen address": {variable: "TIDEGATE_CP__SERVER__LISTEN_ADDRESS=8080", want: `TIDEGATE_CP__SERVER__LISTEN_ADDRESS: invalid listen address "8080", want host:port`},
		"port":           {variable: "TIDEGATE_CP__SERVER__LISTEN_ADDRESS=:99999", want: `TIDEGATE_CP__SERVER__LISTEN_ADDRESS: invalid port in listen address ":99999"`},
		"unknown":        {variable: "TIDEGATE_CP__LOG__LEVL=debug", want: "TIDEGATE_CP__LOG__LEVL: unknown setting"},
		"too deep":       {variable: "TIDEGATE_CP__LOG__LEVEL__X=debug", want: "TIDEGATE_CP__LOG__LEVEL__X: unknown setting"},
		"single level":   {variable: "TIDEGATE_CP__LOG_LEVEL=debug", want: "TIDEGATE_CP__LOG_LEVEL: unknown setting"},
		"prefix only":    {variable: "TIDEGATE_CP__=debug", want: "TIDEGATE_CP__: unknown setting"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			_, err := config.Load([]string{tt.variable})

			// assert
			assert.EqualError(t, err, tt.want)
		})
	}
}

func defaultConfig() config.Config {
	return config.Config{
		Server:  config.Server{ListenAddress: "127.0.0.1:8080"},
		Metrics: config.Metrics{ListenAddress: "127.0.0.1:9464"},
		Log:     config.Log{Level: slog.LevelInfo, Format: logging.FormatJSON},
	}
}
