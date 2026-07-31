package config

import (
	"os"
	"testing"
)

func TestLoad_ValidationJWTSecret(t *testing.T) {
	os.Unsetenv("JWT_SECRET")
	os.Setenv("JWT_SECRET", "short")
	defer os.Unsetenv("JWT_SECRET")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for short JWT_SECRET, got nil")
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	os.Setenv("JWT_SECRET", "12345678901234567890123456789012") // 32 chars
	defer os.Unsetenv("JWT_SECRET")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}

	if cfg.AppPort == "" {
		t.Error("AppPort default should not be empty")
	}
}

func TestLoad_ProductionCheck(t *testing.T) {
	os.Setenv("JWT_SECRET", "12345678901234567890123456789012")
	os.Setenv("APP_ENV", "production")
	os.Unsetenv("MYSQL_DSN")
	defer os.Unsetenv("JWT_SECRET")
	defer os.Unsetenv("APP_ENV")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error in production when MYSQL_DSN is empty")
	}
}
