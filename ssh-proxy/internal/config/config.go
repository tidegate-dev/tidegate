// Package config loads the ssh-proxy's settings from environment variables.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"

	"github.com/phemera-dev/phemera/internal/logging"
)

const (
	envPrefix = "PHEMERA_SSH_PROXY__"

	listenAddress = "LISTEN_ADDRESS"
	logLevel      = "LOG__LEVEL"
	logFormat     = "LOG__FORMAT"
)

// Config holds the ssh-proxy's settings.
type Config struct {
	// ListenAddress is the host and port that the ssh-proxy accepts
	// connections on.
	ListenAddress string
	Log           Log
}

// Log holds the logging settings.
type Log struct {
	Level  slog.Level
	Format logging.Format
}

// Load reads the settings from environ, in the form os.Environ returns. An
// empty variable counts as unset, so the setting keeps its default. The
// error names every variable that is invalid or isn't a setting.
func Load(environ []string) (Config, error) {
	values := map[string]string{
		listenAddress: "127.0.0.1:2022",
		logLevel:      "info",
		logFormat:     string(logging.FormatJSON),
	}

	var errs []error

	for _, entry := range environ {
		name, value, _ := strings.Cut(entry, "=")

		setting, found := strings.CutPrefix(name, envPrefix)
		if !found {
			continue
		}

		setting = strings.ToUpper(setting)

		_, known := values[setting]
		if !known {
			errs = append(errs, fmt.Errorf("%s: unknown setting", name))

			continue
		}

		if value != "" {
			values[setting] = value
		}
	}

	check := func(setting string, err error) {
		if err != nil {
			errs = append(errs, fmt.Errorf("%s%s: %w", envPrefix, setting, err))
		}
	}

	check(listenAddress, checkListenAddress(values[listenAddress]))

	level, err := parseLogLevel(values[logLevel])
	check(logLevel, err)

	format, err := parseLogFormat(values[logFormat])
	check(logFormat, err)

	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}

	return Config{
		ListenAddress: values[listenAddress],
		Log:           Log{Level: level, Format: format},
	}, nil
}

func checkListenAddress(value string) error {
	_, port, err := net.SplitHostPort(value)
	if err != nil {
		return fmt.Errorf("invalid listen address %q, want host:port", value)
	}

	_, err = strconv.ParseUint(port, 10, 16)
	if err != nil {
		return fmt.Errorf("invalid port in listen address %q", value)
	}

	return nil
}

var logLevels = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

func parseLogLevel(value string) (slog.Level, error) {
	level, ok := logLevels[strings.ToLower(value)]
	if !ok {
		return 0, fmt.Errorf("unknown log level %q, want debug, info, warn or error", value)
	}

	return level, nil
}

func parseLogFormat(value string) (logging.Format, error) {
	switch format := logging.Format(strings.ToLower(value)); format {
	case logging.FormatJSON, logging.FormatText:
		return format, nil
	default:
		return "", fmt.Errorf("unknown log format %q, want json or text", value)
	}
}
