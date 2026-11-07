package config

import (
	"log/slog"
	"testing"
	"time"
)

func TestLoadUsesDefaults(t *testing.T) {
	t.Parallel()

	got, err := load(mapEnvironment(map[string]string{"DEMO_MODE": "true"}))
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}

	if got.Environment != defaultEnvironment {
		t.Errorf("Environment = %q, want %q", got.Environment, defaultEnvironment)
	}
	if got.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want %v", got.LogLevel, slog.LevelInfo)
	}
	if got.HTTP.Address != defaultAPIAddress {
		t.Errorf("HTTP.Address = %q, want %q", got.HTTP.Address, defaultAPIAddress)
	}
	if got.HTTP.ReadHeaderTimeout != defaultReadHeaderTimeout {
		t.Errorf("HTTP.ReadHeaderTimeout = %v, want %v", got.HTTP.ReadHeaderTimeout, defaultReadHeaderTimeout)
	}
	if got.HTTP.ReadTimeout != defaultReadTimeout {
		t.Errorf("HTTP.ReadTimeout = %v, want %v", got.HTTP.ReadTimeout, defaultReadTimeout)
	}
	if got.HTTP.WriteTimeout != defaultWriteTimeout {
		t.Errorf("HTTP.WriteTimeout = %v, want %v", got.HTTP.WriteTimeout, defaultWriteTimeout)
	}
	if got.HTTP.IdleTimeout != defaultIdleTimeout {
		t.Errorf("HTTP.IdleTimeout = %v, want %v", got.HTTP.IdleTimeout, defaultIdleTimeout)
	}
	if got.ShutdownTimeout != defaultShutdownTimeout {
		t.Errorf("ShutdownTimeout = %v, want %v", got.ShutdownTimeout, defaultShutdownTimeout)
	}
	if got.Commerce.ManualPaymentEnabled {
		t.Error("ManualPaymentEnabled = true, want false by default")
	}
	if got.SMS.Provider != "log" {
		t.Errorf("SMS.Provider = %q, want log", got.SMS.Provider)
	}
}

func TestLoadReadsEnvironment(t *testing.T) {
	t.Parallel()

	environment := map[string]string{
		"APP_ENV":                     "test",
		"API_ADDR":                    "127.0.0.1:9090",
		"LOG_LEVEL":                   "debug",
		"HTTP_READ_HEADER_TIMEOUT":    "1s",
		"HTTP_READ_TIMEOUT":           "2s",
		"HTTP_WRITE_TIMEOUT":          "3s",
		"HTTP_IDLE_TIMEOUT":           "4s",
		"SHUTDOWN_TIMEOUT":            "5s",
		"DEMO_MODE":                   "true",
		"PAYMENT_MOCK_WEBHOOK_SECRET": "test-mock-webhook-secret-32-bytes!!",
		"JWT_SECRET":                  "test-secret-with-at-least-32-bytes!!",
		"VOUCHER_QR_SECRET":           "test-qr-secret-with-at-least-32-bytes",
	}

	got, err := load(mapEnvironment(environment))
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}

	if got.Environment != "test" {
		t.Errorf("Environment = %q, want test", got.Environment)
	}
	if got.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v, want %v", got.LogLevel, slog.LevelDebug)
	}
	if got.HTTP.Address != "127.0.0.1:9090" {
		t.Errorf("HTTP.Address = %q, want 127.0.0.1:9090", got.HTTP.Address)
	}
	if got.HTTP.ReadHeaderTimeout != time.Second {
		t.Errorf("HTTP.ReadHeaderTimeout = %v, want 1s", got.HTTP.ReadHeaderTimeout)
	}
	if got.HTTP.ReadTimeout != 2*time.Second {
		t.Errorf("HTTP.ReadTimeout = %v, want 2s", got.HTTP.ReadTimeout)
	}
	if got.HTTP.WriteTimeout != 3*time.Second {
		t.Errorf("HTTP.WriteTimeout = %v, want 3s", got.HTTP.WriteTimeout)
	}
	if got.HTTP.IdleTimeout != 4*time.Second {
		t.Errorf("HTTP.IdleTimeout = %v, want 4s", got.HTTP.IdleTimeout)
	}
	if got.ShutdownTimeout != 5*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 5s", got.ShutdownTimeout)
	}
}

func TestLoadRejectsInvalidEnvironment(t *testing.T) {
	t.Parallel()

	tests := map[string]map[string]string{
		"empty environment":        {"APP_ENV": " "},
		"empty address":            {"API_ADDR": ""},
		"invalid log level":        {"LOG_LEVEL": "verbose"},
		"unknown SMS provider":     {"SMS_PROVIDER": "pilot-sms"},
		"log SMS outside demo":     {"SMS_PROVIDER": "log", "DEMO_MODE": "false"},
		"production without redis": {"APP_ENV": "production", "DATABASE_URL": "postgres://x", "JWT_SECRET": "0123456789abcdef0123456789abcdef", "DEMO_MODE": "true"},
		"invalid duration":         {"HTTP_READ_TIMEOUT": "eventually"},
		"non-positive duration":    {"HTTP_IDLE_TIMEOUT": "0s"},
		"negative shutdown period": {"SHUTDOWN_TIMEOUT": "-1s"},
	}

	for name, environment := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := load(mapEnvironment(environment)); err == nil {
				t.Fatal("load() error = nil, want validation error")
			}
		})
	}
}

func emptyEnvironment(string) (string, bool) {
	return "", false
}

func mapEnvironment(values map[string]string) lookupEnv {
	return func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}
}
