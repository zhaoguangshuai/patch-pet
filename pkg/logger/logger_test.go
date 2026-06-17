package logger

import (
	"testing"

	"go.uber.org/zap"
)

func TestIsSensitive(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		expected bool
	}{
		{"password field", "password", true},
		{"passwd field", "passwd", true},
		{"secret field", "secret", true},
		{"token field", "token", true},
		{"api_key field", "api_key", true},
		{"apikey field", "apikey", true},
		{"authorization field", "Authorization", true},
		{"phone field", "phone", true},
		{"mobile field", "mobile", true},
		{"id_card field", "id_card", true},
		{"bank_card field", "bank_card", true},
		{"address field", "address", true},
		{"sql field", "sql", true},
		{"query field", "query", true},
		{"normal field", "user_id", false},
		{"normal field 2", "name", false},
		{"normal field 3", "status", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSensitive(tt.field)
			if result != tt.expected {
				t.Errorf("IsSensitive(%q) = %v, want %v", tt.field, result, tt.expected)
			}
		})
	}
}

func TestSanitizeField(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		expected string
	}{
		{"password", "password", "mysecret", "***REDACTED***"},
		{"token", "token", "abc123xyz", "***REDACTED***"},
		{"normal", "user_id", "12345", "12345"},
		{"phone", "phone", "13800138000", "***REDACTED***"},
		{"address", "address", "北京市朝阳区", "***REDACTED***"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeField(tt.key, tt.value)
			if result != tt.expected {
				t.Errorf("SanitizeField(%q, %q) = %q, want %q", tt.key, tt.value, result, tt.expected)
			}
		})
	}
}

func TestWithTraceID(t *testing.T) {
	// 确保 WithTraceID 不 panic
	logger := WithTraceID("test_trace_id")
	if logger == nil {
		t.Fatal("WithTraceID returned nil")
	}
}

func TestProductionConfig(t *testing.T) {
	cfg := ProductionConfig()
	if cfg.Encoding != "json" {
		t.Errorf("Encoding = %q, want %q", cfg.Encoding, "json")
	}
}

func TestDevelopmentConfig(t *testing.T) {
	cfg := DevelopmentConfig()
	if cfg.Encoding != "console" {
		t.Errorf("Encoding = %q, want %q", cfg.Encoding, "console")
	}
}

func TestInit(t *testing.T) {
	cfg := ProductionConfig()
	err := Init(cfg)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// 验证 logger 可用
	Info("test message", zap.String("key", "value"))
}

func TestGetLogger(t *testing.T) {
	logger := GetLogger()
	if logger == nil {
		t.Fatal("GetLogger returned nil")
	}
}
