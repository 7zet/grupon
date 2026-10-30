package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

const (
	defaultEnvironment       = "development"
	defaultAPIAddress        = ":8080"
	defaultReadHeaderTimeout = 5 * time.Second
	defaultReadTimeout       = 15 * time.Second
	defaultWriteTimeout      = 30 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	defaultShutdownTimeout   = 10 * time.Second
)

// Config contains process configuration shared by the API and worker binaries.
type Config struct {
	Environment     string
	LogLevel        slog.Level
	HTTP            HTTPConfig
	ShutdownTimeout time.Duration
}

// HTTPConfig contains the public HTTP server settings.
type HTTPConfig struct {
	Address           string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

// Load reads configuration from the environment and validates every value.
func Load() (Config, error) {
	return load(os.LookupEnv)
}

type lookupEnv func(string) (string, bool)

func load(lookup lookupEnv) (Config, error) {
	environment, err := stringValue(lookup, "APP_ENV", defaultEnvironment)
	if err != nil {
		return Config{}, err
	}

	address, err := stringValue(lookup, "API_ADDR", defaultAPIAddress)
	if err != nil {
		return Config{}, err
	}

	logLevel, err := levelValue(lookup, "LOG_LEVEL", slog.LevelInfo)
	if err != nil {
		return Config{}, err
	}

	readHeaderTimeout, err := durationValue(lookup, "HTTP_READ_HEADER_TIMEOUT", defaultReadHeaderTimeout)
	if err != nil {
		return Config{}, err
	}

	readTimeout, err := durationValue(lookup, "HTTP_READ_TIMEOUT", defaultReadTimeout)
	if err != nil {
		return Config{}, err
	}

	writeTimeout, err := durationValue(lookup, "HTTP_WRITE_TIMEOUT", defaultWriteTimeout)
	if err != nil {
		return Config{}, err
	}

	idleTimeout, err := durationValue(lookup, "HTTP_IDLE_TIMEOUT", defaultIdleTimeout)
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := durationValue(lookup, "SHUTDOWN_TIMEOUT", defaultShutdownTimeout)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Environment: environment,
		LogLevel:    logLevel,
		HTTP: HTTPConfig{
			Address:           address,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
		ShutdownTimeout: shutdownTimeout,
	}, nil
}

func stringValue(lookup lookupEnv, name, fallback string) (string, error) {
	value, ok := lookup(name)
	if !ok {
		return fallback, nil
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s must not be empty", name)
	}

	return value, nil
}

func levelValue(lookup lookupEnv, name string, fallback slog.Level) (slog.Level, error) {
	value, ok := lookup(name)
	if !ok {
		return fallback, nil
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("%s must not be empty", name)
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(value)); err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}

	return level, nil
}

func durationValue(lookup lookupEnv, name string, fallback time.Duration) (time.Duration, error) {
	value, ok := lookup(name)
	if !ok {
		return fallback, nil
	}

	value = strings.TrimSpace(value)
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
	}

	return duration, nil
}
