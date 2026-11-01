package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadAuthRequiresSecretOutsideDevelopment(t *testing.T) {
	t.Parallel()

	_, err := load(mapEnvironment(map[string]string{"APP_ENV": "production", "DATABASE_URL": "postgres://x"}))
	if err == nil || !strings.Contains(err.Error(), "JWT_SECRET must be set") {
		t.Fatalf("load() error = %v, want JWT_SECRET requirement", err)
	}
}

func TestLoadAuthRejectsShortSecret(t *testing.T) {
	t.Parallel()

	_, err := load(mapEnvironment(map[string]string{"JWT_SECRET": "short"}))
	if err == nil || !strings.Contains(err.Error(), "at least 32 bytes") {
		t.Fatalf("load() error = %v, want length error", err)
	}
}

func TestLoadAuthDefaultsInDevelopment(t *testing.T) {
	t.Parallel()

	got, err := load(mapEnvironment(nil))
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}
	if got.Auth.AccessTokenTTL != 15*time.Minute {
		t.Errorf("AccessTokenTTL = %v, want 15m", got.Auth.AccessTokenTTL)
	}
	if got.Auth.RefreshTokenTTL != 30*24*time.Hour {
		t.Errorf("RefreshTokenTTL = %v, want 720h", got.Auth.RefreshTokenTTL)
	}
	if got.SMTP.Host != "" || got.SMTP.Port != 1025 {
		t.Errorf("SMTP = %+v, want empty host and port 1025", got.SMTP)
	}
}

func TestLoadRejectsInvalidSMTPPort(t *testing.T) {
	t.Parallel()

	_, err := load(mapEnvironment(map[string]string{"SMTP_PORT": "70000"}))
	if err == nil {
		t.Fatal("load() error = nil, want invalid port error")
	}
}
