package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// 有效 26 字符 Crockford Base32 ULID（无 I/L/O/U）
const validULID = "01JZYQ4M8PABCDEFGHJKMNPQRT"

func TestTraceMiddleware_GenerateTraceID(t *testing.T) {
	handler := TraceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := GetTraceID(r.Context())
		if traceID == "" {
			t.Error("trace_id should not be empty")
		}
		if len(traceID) < 3 {
			t.Errorf("trace_id too short: %s", traceID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	respTraceID := rec.Header().Get("X-Trace-Id")
	if respTraceID == "" {
		t.Error("Response X-Trace-Id should not be empty")
	}
}

func TestTraceMiddleware_PassThroughTraceID(t *testing.T) {
	existingTraceID := "trace_" + validULID

	handler := TraceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := GetTraceID(r.Context())
		if traceID != existingTraceID {
			t.Errorf("trace_id = %q, want %q", traceID, existingTraceID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Trace-Id", existingTraceID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	respTraceID := rec.Header().Get("X-Trace-Id")
	if respTraceID != existingTraceID {
		t.Errorf("Response X-Trace-Id = %q, want %q", respTraceID, existingTraceID)
	}
}

func TestHeaderValidationMiddleware_MissingAuthorization(t *testing.T) {
	handler := HeaderValidationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Should not reach handler")
	}))

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHeaderValidationMiddleware_MissingRequestSource(t *testing.T) {
	handler := HeaderValidationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Should not reach handler")
	}))

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer test_token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHeaderValidationMiddleware_ValidHeaders(t *testing.T) {
	handler := HeaderValidationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		source := GetRequestSource(r.Context())
		if source != "app" {
			t.Errorf("request_source = %q, want %q", source, "app")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer test_token")
	req.Header.Set("X-Request-Source", "app")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHeaderValidationMiddleware_HealthCheckSkip(t *testing.T) {
	handler := HeaderValidationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestGetTraceID_EmptyContext(t *testing.T) {
	ctx := context.Background()
	traceID := GetTraceID(ctx)
	if traceID != "" {
		t.Errorf("trace_id = %q, want empty", traceID)
	}
}

func TestGetUserID_EmptyContext(t *testing.T) {
	ctx := context.Background()
	userID := GetUserID(ctx)
	if userID != "" {
		t.Errorf("user_id = %q, want empty", userID)
	}
}

func TestGetRequestSource_EmptyContext(t *testing.T) {
	ctx := context.Background()
	source := GetRequestSource(ctx)
	if source != "" {
		t.Errorf("request_source = %q, want empty", source)
	}
}

func TestIsValidULIDFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid trace prefix", "trace_" + validULID, true},
		{"valid pure ULID", validULID, true},
		{"too short", "01JZYQ4M8P", false},
		{"too long", validULID + "0000", false},
		{"invalid char I", "01JZYQ4M8PABCDEFGHJKLMNPQI", false},
		{"invalid char L", "01JZYQ4M8PABCDEFGHJKLMNPQL", false},
		{"invalid char O", "01JZYQ4M8PABCDEFGHJKLMNPQO", false},
		{"invalid char U", "01JZYQ4M8PABCDEFGHJKLMNPQU", false},
		{"empty", "", false},
		{"wrong prefix", "trace0_" + validULID, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidULIDFormat(tt.input)
			if result != tt.expected {
				t.Errorf("isValidULIDFormat(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
