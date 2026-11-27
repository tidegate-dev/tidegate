// Package config loads the authority's settings from environment
// variables.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"

	"github.com/phemera-dev/phemera/internal/dburl"
	"github.com/phemera-dev/phemera/internal/logging"
)

const (
	envPrefix      = "PHEMERA_AUTHORITY__"
	levelSeparator = "__"
	structTag      = "koanf"
)

// rawConfig keeps every setting as a string, because environment variables
// arrive as strings and validate needs the original value for its errors.
type rawConfig struct {
	ExternalAPI rawExternalAPI `koanf:"external_api"`
	InternalAPI rawInternalAPI `koanf:"internal_api"`
	Metrics     rawMetrics     `koanf:"metrics"`
	Database    rawDatabase    `koanf:"database"`
	Login       rawLogin       `koanf:"login"`
	Log         rawLog         `koanf:"log"`
}

type rawExternalAPI struct {
	ListenAddress string         `koanf:"listen_address"`
	TLS           rawExternalTLS `koanf:"tls"`
}

type rawExternalTLS struct {
	Disabled string `koanf:"disabled"`
	CertFile string `koanf:"cert_file"`
	KeyFile  string `koanf:"key_file"`
}

type rawInternalAPI struct {
	ListenAddress string         `koanf:"listen_address"`
	TLS           rawInternalTLS `koanf:"tls"`
}

type rawInternalTLS struct {
	CertFile string `koanf:"cert_file"`
	KeyFile  string `koanf:"key_file"`
}

type rawMetrics struct {
	ListenAddress string `koanf:"listen_address"`
}

type rawDatabase struct {
	URL            string `koanf:"url"`
	MigrateOnStart string `koanf:"migrate_on_start"`
}

type rawLogin struct {
	TokenLifetime string `koanf:"token_lifetime"`
}

type rawLog struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"`
}

// tls.crt and tls.key are the file names in a Kubernetes TLS Secret.
var defaults = rawConfig{
	ExternalAPI: rawExternalAPI{
		ListenAddress: "127.0.0.1:8080",
		TLS: rawExternalTLS{
			Disabled: "false",
			CertFile: "/etc/phemera/authority/external-api-tls/tls.crt",
			KeyFile:  "/etc/phemera/authority/external-api-tls/tls.key",
		},
	},
	InternalAPI: rawInternalAPI{
		ListenAddress: "127.0.0.1:8081",
		TLS: rawInternalTLS{
			CertFile: "/etc/phemera/authority/internal-api-tls/tls.crt",
			KeyFile:  "/etc/phemera/authority/internal-api-tls/tls.key",
		},
	},
	Metrics: rawMetrics{
		ListenAddress: "127.0.0.1:9464",
	},
	Database: rawDatabase{
		MigrateOnStart: "false",
	},
	Login: rawLogin{
		TokenLifetime: "12h",
	},
	Log: rawLog{
		Level:  "info",
		Format: string(logging.FormatJSON),
	},
}

// Config holds the authority's settings.
type Config struct {
	ExternalAPI ExternalAPI
	InternalAPI InternalAPI
	Metrics     Metrics
	Database    Database
	Login       Login
	Log         Log
}

// ExternalAPI holds the settings of the server for the external API, which
// the CLI and the browser call.
type ExternalAPI struct {
	ListenAddress string
	TLS           ExternalTLS
}

// ExternalTLS holds where the external API finds its certificate, or that
// it serves plain HTTP.
type ExternalTLS struct {
	Disabled bool
	CertFile string
	KeyFile  string
}

// InternalAPI holds the settings of the server for the internal API, which
// gateways and workers call.
type InternalAPI struct {
	ListenAddress string
	TLS           InternalTLS
}

// InternalTLS holds where the internal API finds its certificate. It can't
// be disabled, because a proxy that terminates TLS hides the client's
// certificate.
type InternalTLS struct {
	CertFile string
	KeyFile  string
}

// Metrics holds the settings of the server that serves the metrics.
type Metrics struct {
	ListenAddress string
}

// Database holds the settings for the connection to Postgres.
type Database struct {
	URL            string
	MigrateOnStart bool
}

// Login holds the login settings.
type Login struct {
	TokenLifetime time.Duration
}

// Log holds the logging settings.
type Log struct {
	Level  slog.Level
	Format logging.Format
}

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

	set := make(map[string]bool)

	err = k.Load(env.Provider(".", env.Opt{
		Prefix: envPrefix,
		TransformFunc: func(name, value string) (string, any) {
			key := keyFromVariable(name)
			if !known[key] || value == "" {
				return "", nil
			}

			set[key] = true

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

	cfg, err := validate(raw, set)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}

	return cfg, nil
}

// set holds the keys that a variable set, as opposed to a default.
func validate(raw rawConfig, set map[string]bool) (Config, error) {
	v := validator{keys: fieldKeys(&raw), set: set}

	externalListenAddress, err := parseListenAddress(raw.ExternalAPI.ListenAddress)
	v.check(&raw.ExternalAPI.ListenAddress, err)

	externalTLS := v.externalTLS(&raw.ExternalAPI.TLS)

	internalListenAddress, err := parseListenAddress(raw.InternalAPI.ListenAddress)
	v.check(&raw.InternalAPI.ListenAddress, err)

	metricsListenAddress, err := parseListenAddress(raw.Metrics.ListenAddress)
	v.check(&raw.Metrics.ListenAddress, err)

	v.checkOverlaps(&raw.ExternalAPI.ListenAddress, &raw.InternalAPI.ListenAddress, &raw.Metrics.ListenAddress)

	v.check(&raw.Database.URL, checkDatabaseURL(raw.Database.URL))

	migrateOnStart, err := parseBool(raw.Database.MigrateOnStart)
	v.check(&raw.Database.MigrateOnStart, err)

	tokenLifetime, err := parseLifetime(raw.Login.TokenLifetime)
	v.check(&raw.Login.TokenLifetime, err)

	level, err := parseLogLevel(raw.Log.Level)
	v.check(&raw.Log.Level, err)

	format, err := parseLogFormat(raw.Log.Format)
	v.check(&raw.Log.Format, err)

	if len(v.errs) > 0 {
		return Config{}, errors.Join(v.errs...)
	}

	return Config{
		ExternalAPI: ExternalAPI{ListenAddress: externalListenAddress, TLS: externalTLS},
		InternalAPI: InternalAPI{
			ListenAddress: internalListenAddress,
			TLS:           InternalTLS{CertFile: raw.InternalAPI.TLS.CertFile, KeyFile: raw.InternalAPI.TLS.KeyFile},
		},
		Metrics:  Metrics{ListenAddress: metricsListenAddress},
		Database: Database{URL: raw.Database.URL, MigrateOnStart: migrateOnStart},
		Login:    Login{TokenLifetime: tokenLifetime},
		Log:      Log{Level: level, Format: format},
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
	set  map[string]bool
	errs []error
}

// check takes the field instead of its key, so validate doesn't repeat the
// keys that the koanf tags already define.
func (v *validator) check(field *string, err error) {
	if err != nil {
		v.errs = append(v.errs, fmt.Errorf("%s: %w", variableFromKey(v.keys[field]), err))
	}
}

func (v *validator) checkOverlaps(listenAddresses ...*string) {
	for i, address := range listenAddresses {
		for _, earlier := range listenAddresses[:i] {
			if overlap(*address, *earlier) {
				v.check(address, fmt.Errorf("%q overlaps with %q in %s", *address, *earlier, variableFromKey(v.keys[earlier])))
			}
		}
	}
}

func (v *validator) externalTLS(raw *rawExternalTLS) ExternalTLS {
	disabled, err := parseBool(raw.Disabled)
	v.check(&raw.Disabled, err)

	if disabled && (v.set[v.keys[&raw.CertFile]] || v.set[v.keys[&raw.KeyFile]]) {
		v.check(&raw.Disabled, fmt.Errorf("can't be true while %s or %s is set",
			variableFromKey(v.keys[&raw.CertFile]), variableFromKey(v.keys[&raw.KeyFile])))
	}

	return ExternalTLS{Disabled: disabled, CertFile: raw.CertFile, KeyFile: raw.KeyFile}
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

// ExternalTLSDisabledVariable returns the name of the variable that turns
// off TLS for the external API, for messages that tell an operator to set it.
func ExternalTLSDisabledVariable() string {
	var raw rawConfig

	return variableFromKey(fieldKeys(&raw)[&raw.ExternalAPI.TLS.Disabled])
}

func variableFromKey(key string) string {
	return envPrefix + strings.ToUpper(strings.ReplaceAll(key, ".", levelSeparator))
}

func checkDatabaseURL(url string) error {
	if url == "" {
		return errors.New("must be set")
	}

	_, err := dburl.Parse(url)

	return err
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

// overlap reports only what is certain, so it doesn't resolve names.
func overlap(listenAddress, other string) bool {
	host, port, err := net.SplitHostPort(listenAddress)
	if err != nil {
		return false
	}

	otherHost, otherPort, err := net.SplitHostPort(other)
	if err != nil {
		return false
	}

	number, err := strconv.ParseUint(port, 10, 16)
	if err != nil || number == 0 {
		return false
	}

	otherNumber, err := strconv.ParseUint(otherPort, 10, 16)
	if err != nil || number != otherNumber {
		return false
	}

	ip, err := netip.ParseAddr(host)
	otherIP, otherErr := netip.ParseAddr(otherHost)

	// Go binds an unspecified address to IPv4 and IPv6 where the system has
	// dual-stack sockets, as Linux does.
	if host == "" || otherHost == "" || ip.IsUnspecified() || otherIP.IsUnspecified() {
		return true
	}

	if err == nil && otherErr == nil {
		return ip.Unmap() == otherIP.Unmap()
	}

	return strings.EqualFold(host, otherHost)
}

func parseBool(value string) (bool, error) {
	switch strings.ToLower(value) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("invalid value %q, want true or false", value)
	}
}

func parseLifetime(value string) (time.Duration, error) {
	lifetime, err := time.ParseDuration(strings.ToLower(value))
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q, want a number with a unit, like 12h or 30m", value)
	}

	// Postgres rounds intervals to microseconds, and a shorter token is useless.
	if lifetime < time.Second {
		return 0, errors.New("must be at least 1s")
	}

	return lifetime, nil
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
