package thirdparty

import (
	"os"
	"testing"
)

func TestDefaultRedisConfig(t *testing.T) {
	os.Setenv("REDIS_ADDR", "redis-test:6379")
	os.Setenv("REDIS_PASSWORD", "test-pass")
	defer os.Unsetenv("REDIS_ADDR")
	defer os.Unsetenv("REDIS_PASSWORD")

	cfg := DefaultRedisConfig()

	if cfg.Addr != "redis-test:6379" {
		t.Errorf("Addr = %q, want %q", cfg.Addr, "redis-test:6379")
	}
	if cfg.Password != "test-pass" {
		t.Errorf("Password = %q, want %q", cfg.Password, "test-pass")
	}
	if cfg.DB != 0 {
		t.Errorf("DB = %d, want 0", cfg.DB)
	}
}

func TestDefaultRedisConfigDefaults(t *testing.T) {
	os.Unsetenv("REDIS_ADDR")
	os.Unsetenv("REDIS_PASSWORD")

	cfg := DefaultRedisConfig()

	if cfg.Addr != "localhost:6379" {
		t.Errorf("Addr = %q, want %q", cfg.Addr, "localhost:6379")
	}
	if cfg.Password != "" {
		t.Errorf("Password = %q, want empty", cfg.Password)
	}
}

func TestNewRedisClientCreation(t *testing.T) {
	cfg := RedisConfig{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}

	client := NewRedisClient(cfg)
	if client == nil {
		t.Fatal("NewRedisClient should not return nil")
	}
}

func TestLockAcquiredDefault(t *testing.T) {
	l := &Lock{acquired: false}
	if l.Acquired() {
		t.Error("new Lock should not be acquired")
	}
}
