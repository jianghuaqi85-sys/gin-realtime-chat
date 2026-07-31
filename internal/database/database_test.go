package database

import (
	"testing"
	"time"
)

func TestConnectWithConfig_InvalidDSNFailFast(t *testing.T) {
	cfg := &DBConfig{
		DSN:             "invalid_user:invalid_pass@tcp(127.0.0.1:9999)/invalid_db?timeout=1s",
		LogLevel:        "silent",
		MaxIdleConns:    5,
		MaxOpenConns:    10,
		ConnMaxIdleTime: time.Minute,
		ConnMaxLifetime: time.Hour,
	}

	_, err := ConnectWithConfig(cfg)
	if err == nil {
		t.Fatal("expected error for invalid DSN/host, got nil (fail-fast ping failed)")
	}
}
