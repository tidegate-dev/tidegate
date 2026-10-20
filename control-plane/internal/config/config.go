// Package config loads the control plane's settings from environment
// variables.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"reflect"
	"strconv"
	"strings"

	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

const (
	envPrefix      = "TIDEGATE_CP__"
	levelSeparator = "__"
	structTag      = "koanf"
)

// rawConfig keeps every setting as a string, because environment variables
// arrive as strings and validate needs the original value for its errors.
type rawConfig struct {
	Server rawServer `koanf:"server"`
	Log    rawLog    `koanf:"log"`
}

type rawServer struct {
	ListenAddress string `koanf:"listen_address"`
}

type rawLog struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"`
}

var defaults = rawConfig{
	Server: rawServer{
		ListenAddress: "127.0.0.1:8080",
	},
	Log: rawLog{
		Level:  "info",
		Format: string(LogFormatJSON),
	},
}

// Config holds the control plane's settings.
type Config struct {
	Server Server
	Log    Log
}

// Server holds the settings of the control plane's API server.
type Server struct {
	ListenAddress string
}

// Log holds the logging settings.
type Log struct {
	Level  slog.Level
	Format LogFormat
}

// LogFormat is how log lines are written.
type LogFormat string

const (
	// LogFormatJSON writes one JSON object per line, for log collectors.
	LogFormatJSON LogFormat = "json"
	// LogFormatText writes key=value pairs, which are easier to read locally.
	LogFormatText LogFormat = "text"
)

// Load reads the settings from environ, in the form os.Environ returns. An
// empty variable counts as unset, so the setting keeps its default.
func Load(environ []string) (Config, error) {
	k := koanf.New(".")

	err := k.Load(structs.Provider(defaults, structTag), nil)
	if err != nil {
		return Config{}, fmt.Errorf("loading defaults: %w", err)
	}

	known := make(map[string]bool)
	for _, key := range k.Keys() {
		known[key] = true
	}

	errs := unknownVariables(environ, known)

	err = k.Load(env.Provider(".", env.Opt{
		Prefix: envPrefix,
		TransformFunc: func(name, value string) (string, any) {
			key := keyFromVariable(name)
			if !known[key] || value == "" {
				return "", nil
			}

			return key, value
		},
		EnvironFunc: func() []string { return environ },
	}), nil)
	if err != nil {
		return Config{}, fmt.Errorf("loading environment variables: %w", err)
	}

	var raw rawConfig

	err = k.UnmarshalWithConf("", &raw, koanf.UnmarshalConf{Tag: structTag})
	if err != nil {
		return Config{}, fmt.Errorf("decoding settings: %w", err)
	}

	cfg, err := validate(raw)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}

	return cfg, nil
}

func validate(raw rawConfig) (Config, error) {
	v := validator{keys: fieldKeys(&raw)}

	listenAddress, err := parseListenAddress(raw.Server.ListenAddress)
	v.check(&raw.Server.ListenAddress, err)

	level, err := parseLogLevel(raw.Log.Level)
	v.check(&raw.Log.Level, err)

	format, err := parseLogFormat(raw.Log.Format)
	v.check(&raw.Log.Format, err)

	if len(v.errs) > 0 {
		return Config{}, errors.Join(v.errs...)
	}

	return Config{
		Server: Server{ListenAddress: listenAddress},
		Log:    Log{Level: level, Format: format},
	}, nil
}

func unknownVariables(environ []string, known map[string]bool) []error {
	var errs []error

	for _, entry := range environ {
		name, _, _ := strings.Cut(entry, "=")
		if !strings.HasPrefix(name, envPrefix) {
			continue
		}

		if !known[keyFromVariable(name)] {
			errs = append(errs, fmt.Errorf("%s: unknown setting", name))
		}
	}

	return errs
}

type validator struct {
	keys map[*string]string
	errs []error
}

// check takes the field instead of its key, so validate doesn't repeat the
// keys that the koanf tags already define.
func (v *validator) check(field *string, err error) {
	if err != nil {
		v.errs = append(v.errs, fmt.Errorf("%s: %w", variableFromKey(v.keys[field]), err))
	}
}

func fieldKeys(raw *rawConfig) map[*string]string {
	keys := make(map[*string]string)

	var walk func(value reflect.Value, prefix string)

	walk = func(value reflect.Value, prefix string) {
		for i := range value.NumField() {
			field := value.Field(i)
			key := prefix + value.Type().Field(i).Tag.Get(structTag)

			switch field.Kind() {
			case reflect.Struct:
				walk(field, key+".")
			case reflect.String:
				keys[field.Addr().Interface().(*string)] = key
			default:
				panic(fmt.Sprintf("config: setting %s must be a string", key))
			}
		}
	}

	walk(reflect.ValueOf(raw).Elem(), "")

	return keys
}

func keyFromVariable(name string) string {
	rest := strings.TrimPrefix(name, envPrefix)

	return strings.ReplaceAll(strings.ToLower(rest), levelSeparator, ".")
}

func variableFromKey(key string) string {
	return envPrefix + strings.ToUpper(strings.ReplaceAll(key, ".", levelSeparator))
}

func parseListenAddress(value string) (string, error) {
	_, port, err := net.SplitHostPort(value)
	if err != nil {
		return "", fmt.Errorf("invalid listen address %q, want host:port", value)
	}

	_, err = strconv.ParseUint(port, 10, 16)
	if err != nil {
		return "", fmt.Errorf("invalid port in listen address %q", value)
	}

	return value, nil
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

func parseLogFormat(value string) (LogFormat, error) {
	switch format := LogFormat(strings.ToLower(value)); format {
	case LogFormatJSON, LogFormatText:
		return format, nil
	default:
		return "", fmt.Errorf("unknown log format %q, want json or text", value)
	}
}
