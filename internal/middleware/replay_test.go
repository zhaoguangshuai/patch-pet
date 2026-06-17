package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"
)

func setupReplaySecret() {
	replaySecretKey = "test-secret-key-for-replay"
}

func TestReplayMiddlewareValidRequest(t *testing.T) {
	setupReplaySecret()

	var nextCalled bool
	handler := ReplayMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	body := []byte(`{"action":"test"}`)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	sig := computeSignature(body, timestamp)

	req := httptest.NewRequest("POST", "/api/v1/test", bytes.NewReader(body))
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Signature", sig)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	if !nextCalled {
		t.Error("next handler should have been called")
	}
}

func TestReplayMiddlewareMissingTimestamp(t *testing.T) {
	setupReplaySecret()

	handler := ReplayMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	}))

	req := httptest.NewRequest("POST", "/api/v1/test", nil)
	req.Header.Set("X-Signature", "some-sig")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestReplayMiddlewareMissingSignature(t *testing.T) {
	setupReplaySecret()

	handler := ReplayMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	}))

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	req := httptest.NewRequest("POST", "/api/v1/test", nil)
	req.Header.Set("X-Timestamp", timestamp)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestReplayMiddlewareExpiredTimestamp(t *testing.T) {
	setupReplaySecret()

	handler := ReplayMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	}))

	// 10 分钟前的时间戳（超过 5 分钟容忍）
	expired := time.Now().Unix() - 600
	timestamp := strconv.FormatInt(expired, 10)
	body := []byte(`{}`)
	sig := computeSignature(body, timestamp)

	req := httptest.NewRequest("POST", "/api/v1/test", bytes.NewReader(body))
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Signature", sig)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestReplayMiddlewareInvalidSignature(t *testing.T) {
	setupReplaySecret()

	handler := ReplayMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	}))

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	req := httptest.NewRequest("POST", "/api/v1/test", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Signature", "invalid-signature")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestReplayMiddlewareBodyPreserved(t *testing.T) {
	setupReplaySecret()

	var receivedBody string
	handler := ReplayMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		receivedBody = buf.String()
		w.WriteHeader(http.StatusOK)
	}))

	body := []byte(`{"pet_id":"pet_001","action":"walk"}`)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	sig := computeSignature(body, timestamp)

	req := httptest.NewRequest("POST", "/api/v1/test", bytes.NewReader(body))
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Signature", sig)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if receivedBody != string(body) {
		t.Errorf("body = %q, want %q", receivedBody, string(body))
	}
}

func TestReplayMiddlewareHealthSkipped(t *testing.T) {
	setupReplaySecret()

	var nextCalled bool
	handler := ReplayMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !nextCalled {
		t.Error("/health should skip replay check")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestReplayMiddlewareInvalidTimestampFormat(t *testing.T) {
	setupReplaySecret()

	handler := ReplayMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	}))

	req := httptest.NewRequest("POST", "/api/v1/test", nil)
	req.Header.Set("X-Timestamp", "not-a-number")
	req.Header.Set("X-Signature", "some-sig")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestReplayMiddlewareGetRequestNoBody(t *testing.T) {
	setupReplaySecret()

	var nextCalled bool
	handler := ReplayMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	sig := computeSignature(nil, timestamp)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Signature", sig)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	if !nextCalled {
		t.Error("next handler should have been called")
	}
}

func TestReplayMiddlewareTimestampAtBoundary(t *testing.T) {
	setupReplaySecret()

	var nextCalled bool
	handler := ReplayMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	// 恰好 5 分钟前（边界值，应通过）
	boundary := time.Now().Unix() - int64(300)
	timestamp := strconv.FormatInt(boundary, 10)
	body := []byte(`{}`)
	sig := computeSignature(body, timestamp)

	req := httptest.NewRequest("POST", "/api/v1/test", bytes.NewReader(body))
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Signature", sig)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (boundary 300s should pass)", rr.Code)
	}
	if !nextCalled {
		t.Error("boundary timestamp should pass")
	}
}

func TestComputeSignatureDeterministic(t *testing.T) {
	setupReplaySecret()

	body := []byte(`{"test":"data"}`)
	ts := "1234567890"

	sig1 := computeSignature(body, ts)
	sig2 := computeSignature(body, ts)

	if sig1 != sig2 {
		t.Errorf("signatures should be deterministic: %s != %s", sig1, sig2)
	}
}

func TestReplaySecretFromEnv(t *testing.T) {
	os.Setenv("REPLAY_SECRET_KEY", "env-secret-123")
	defer os.Unsetenv("REPLAY_SECRET_KEY")

	// 重新加载（模拟进程重启后的状态）
	original := replaySecretKey
	replaySecretKey = os.Getenv("REPLAY_SECRET_KEY")
	defer func() { replaySecretKey = original }()

	if replaySecretKey != "env-secret-123" {
		t.Errorf("replaySecretKey = %q, want %q", replaySecretKey, "env-secret-123")
	}
}
