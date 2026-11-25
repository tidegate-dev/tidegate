package config_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/internal/logging"
	"github.com/tidegate-dev/tidegate/ssh-proxy/internal/config"
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
		"TIDEGATE_SSH_PROXY__LISTEN_ADDRESS=:2200",
		"TIDEGATE_SSH_PROXY__LOG__LEVEL=debug",
		"TIDEGATE_SSH_PROXY__LOG__FORMAT=text",
	}

	// act
	cfg, err := config.Load(environ)

	// assert
	require.NoError(t, err)
	assert.Equal(t, config.Config{
		ListenAddress: ":2200",
		Log:           config.Log{Level: slog.LevelDebug, Format: logging.FormatText},
	}, cfg)
}

func TestNamesAfterThePrefixAndValuesIgnoreLetterCase(t *testing.T) {
	t.Parallel()

	// arrange
	environ := []string{
		"TIDEGATE_SSH_PROXY__log__level=DEBUG",
		"TIDEGATE_SSH_PROXY__Log__Format=Text",
	}

	// act
	cfg, err := config.Load(environ)

	// assert
	require.NoError(t, err)
	assert.Equal(t, config.Log{Level: slog.LevelDebug, Format: logging.FormatText}, cfg.Log)
}

func TestEmptyVariableKeepsTheDefault(t *testing.T) {
	t.Parallel()

	// act
	cfg, err := config.Load([]string{"TIDEGATE_SSH_PROXY__LISTEN_ADDRESS="})

	// assert
	require.NoError(t, err)
	assert.Equal(t, defaultConfig(), cfg)
}

func TestVariablesThatAreNotSettingsAreIgnored(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"without prefix":                    "LOG__LEVEL=loud",
		"the authority's setting":           "TIDEGATE_AUTHORITY__LOG__LEVEL=loud",
		"kubernetes service host":           "TIDEGATE_SSH_PROXY_SERVICE_HOST=10.0.0.1",
		"single underscore after SSH_PROXY": "TIDEGATE_SSH_PROXY_LOG__LEVEL=loud",
		"prefix in lower case":              "tidegate_ssh_proxy__LOG__LEVEL=loud",
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

func TestInvalidVariableFailsWithItsNameAndTheReason(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		variable string
		message  string
	}{
		"unknown setting": {
			variable: "TIDEGATE_SSH_PROXY__LOG__LEVL=debug",
			message:  "TIDEGATE_SSH_PROXY__LOG__LEVL: unknown setting",
		},
		"listen address without a port": {
			variable: "TIDEGATE_SSH_PROXY__LISTEN_ADDRESS=localhost",
			message:  `TIDEGATE_SSH_PROXY__LISTEN_ADDRESS: invalid listen address "localhost", want host:port`,
		},
		"listen address with a port name": {
			variable: "TIDEGATE_SSH_PROXY__LISTEN_ADDRESS=:ssh",
			message:  `TIDEGATE_SSH_PROXY__LISTEN_ADDRESS: invalid port in listen address ":ssh"`,
		},
		"listen address with a port that is too high": {
			variable: "TIDEGATE_SSH_PROXY__LISTEN_ADDRESS=:65536",
			message:  `TIDEGATE_SSH_PROXY__LISTEN_ADDRESS: invalid port in listen address ":65536"`,
		},
		"log level": {
			variable: "TIDEGATE_SSH_PROXY__LOG__LEVEL=loud",
			message:  `TIDEGATE_SSH_PROXY__LOG__LEVEL: unknown log level "loud", want debug, info, warn or error`,
		},
		"log format": {
			variable: "TIDEGATE_SSH_PROXY__LOG__FORMAT=xml",
			message:  `TIDEGATE_SSH_PROXY__LOG__FORMAT: unknown log format "xml", want json or text`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// act
			_, err := config.Load([]string{test.variable})

			// assert
			require.EqualError(t, err, test.message)
		})
	}
}

func TestEveryInvalidVariableIsReported(t *testing.T) {
	t.Parallel()

	// arrange
	environ := []string{
		"TIDEGATE_SSH_PROXY__LOG__LEVEL=loud",
		"TIDEGATE_SSH_PROXY__LOG__FORMAT=xml",
	}

	// act
	_, err := config.Load(environ)

	// assert
	require.ErrorContains(t, err, "TIDEGATE_SSH_PROXY__LOG__LEVEL: ")
	require.ErrorContains(t, err, "TIDEGATE_SSH_PROXY__LOG__FORMAT: ")
}

func defaultConfig() config.Config {
	return config.Config{
		ListenAddress: "127.0.0.1:2022",
		Log:           config.Log{Level: slog.LevelInfo, Format: logging.FormatJSON},
	}
}
