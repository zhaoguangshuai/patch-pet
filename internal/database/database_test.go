package database

import (
	"os"
	"testing"
)

func TestDefaultDBConfig(t *testing.T) {
	os.Setenv("POSTGRES_DSN", "host=localhost user=test dbname=testdb")
	defer os.Unsetenv("POSTGRES_DSN")

	cfg := DefaultDBConfig()

	if cfg.DSN != "host=localhost user=test dbname=testdb" {
		t.Errorf("DSN = %q, want from env", cfg.DSN)
	}
	if cfg.MaxOpenConns != 25 {
		t.Errorf("MaxOpenConns = %d, want 25", cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != 10 {
		t.Errorf("MaxIdleConns = %d, want 10", cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime != 300 {
		t.Errorf("ConnMaxLifetime = %d, want 300", cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime != 60 {
		t.Errorf("ConnMaxIdleTime = %d, want 60", cfg.ConnMaxIdleTime)
	}
}

func TestDefaultDBConfigEmptyDSN(t *testing.T) {
	os.Unsetenv("POSTGRES_DSN")

	cfg := DefaultDBConfig()

	if cfg.DSN != "" {
		t.Errorf("DSN = %q, want empty when env not set", cfg.DSN)
	}
}

func TestNewWithEmptyDSN(t *testing.T) {
	_, err := New(DBConfig{DSN: ""})
	if err == nil {
		t.Error("expected error for empty DSN")
	}
}

func TestNewWithInvalidDSN(t *testing.T) {
	_, err := New(DBConfig{DSN: "invalid-dsn-no-host"})
	if err == nil {
		t.Error("expected error for invalid DSN")
	}
}
