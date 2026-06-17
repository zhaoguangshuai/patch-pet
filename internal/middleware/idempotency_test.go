package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIdempotencyMiddlewarePostRequiresKey(t *testing.T) {
	store := NewMemoryIdempotencyStore()
	mw := IdempotencyMiddleware(store)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))

	req := httptest.NewRequest("POST", "/api/v1/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (missing Idempotency-Key)", rr.Code)
	}
}

func TestIdempotencyMiddlewareDuplicateRequest(t *testing.T) {
	store := NewMemoryIdempotencyStore()
	mw := IdempotencyMiddleware(store)

	callCount := 0
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"order_001"}`))
	}))

	// 第一次请求
	req1 := httptest.NewRequest("POST", "/api/v1/orders", strings.NewReader(`{}`))
	req1.Header.Set("Idempotency-Key", "key-001")
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusCreated {
		t.Errorf("first request: status = %d, want 201", rr1.Code)
	}
	if callCount != 1 {
		t.Errorf("first request: callCount = %d, want 1", callCount)
	}

	// 第二次请求（相同 key）
	req2 := httptest.NewRequest("POST", "/api/v1/orders", strings.NewReader(`{}`))
	req2.Header.Set("Idempotency-Key", "key-001")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusCreated {
		t.Errorf("second request: status = %d, want 201", rr2.Code)
	}
	if callCount != 1 {
		t.Errorf("second request: callCount = %d, want 1 (handler should not be called again)", callCount)
	}
	if rr2.Header().Get("X-Idempotent-Replay") != "true" {
		t.Error("second request should have X-Idempotent-Replay header")
	}
}

func TestIdempotencyMiddlewareDifferentKeys(t *testing.T) {
	store := NewMemoryIdempotencyStore()
	mw := IdempotencyMiddleware(store)

	callCount := 0
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest("POST", "/api/v1/test", nil)
	req1.Header.Set("Idempotency-Key", "key-A")
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	req2 := httptest.NewRequest("POST", "/api/v1/test", nil)
	req2.Header.Set("Idempotency-Key", "key-B")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if callCount != 2 {
		t.Errorf("callCount = %d, want 2 (different keys)", callCount)
	}
}

func TestIdempotencyMiddlewareGetSkipped(t *testing.T) {
	store := NewMemoryIdempotencyStore()
	mw := IdempotencyMiddleware(store)

	var nextCalled bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !nextCalled {
		t.Error("GET request should skip idempotency check")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("GET status = %d, want 200", rr.Code)
	}
}

func TestIdempotencyMiddlewarePutRequiresKey(t *testing.T) {
	store := NewMemoryIdempotencyStore()
	mw := IdempotencyMiddleware(store)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("PUT", "/api/v1/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("PUT without key: status = %d, want 400", rr.Code)
	}
}

func TestIdempotencyMiddlewareErrorResponseNotCached(t *testing.T) {
	store := NewMemoryIdempotencyStore()
	mw := IdempotencyMiddleware(store)

	callCount := 0
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"fail"}`))
	}))

	// 第一次请求（500 错误）
	req1 := httptest.NewRequest("POST", "/api/v1/test", nil)
	req1.Header.Set("Idempotency-Key", "err-key")
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	// 第二次请求（相同 key，应重新执行）
	req2 := httptest.NewRequest("POST", "/api/v1/test", nil)
	req2.Header.Set("Idempotency-Key", "err-key")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if callCount != 2 {
		t.Errorf("callCount = %d, want 2 (error responses should not be cached)", callCount)
	}
}

func TestIdempotencyMiddlewareHealthSkipped(t *testing.T) {
	store := NewMemoryIdempotencyStore()
	mw := IdempotencyMiddleware(store)

	var nextCalled bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !nextCalled {
		t.Error("/health should skip idempotency check")
	}
}

func TestIdempotencyMiddlewareResponsePreserved(t *testing.T) {
	store := NewMemoryIdempotencyStore()
	mw := IdempotencyMiddleware(store)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "value")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"order_123"}`))
	}))

	// 第一次
	req1 := httptest.NewRequest("POST", "/api/v1/test", nil)
	req1.Header.Set("Idempotency-Key", "preserve-key")
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	// 第二次
	req2 := httptest.NewRequest("POST", "/api/v1/test", nil)
	req2.Header.Set("Idempotency-Key", "preserve-key")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	// 验证缓存的响应完整保留
	if rr2.Code != http.StatusCreated {
		t.Errorf("cached status = %d, want 201", rr2.Code)
	}
	if rr2.Body.String() != `{"id":"order_123"}` {
		t.Errorf("cached body = %q, want %q", rr2.Body.String(), `{"id":"order_123"}`)
	}
}

// ---- MemoryIdempotencyStore ----

func TestMemoryStoreSetGet(t *testing.T) {
	store := NewMemoryIdempotencyStore()

	resp := &CachedResponse{StatusCode: 200, Body: []byte("ok"), CreatedAt: time.Now()}
	store.Set("k1", resp, 0)

	got, ok := store.Get("k1")
	if !ok {
		t.Fatal("Get should return true")
	}
	if got.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", got.StatusCode)
	}
}

func TestMemoryStoreDelete(t *testing.T) {
	store := NewMemoryIdempotencyStore()

	store.Set("k1", &CachedResponse{StatusCode: 200, CreatedAt: time.Now()}, 0)
	store.Delete("k1")

	_, ok := store.Get("k1")
	if ok {
		t.Error("Get after Delete should return false")
	}
}

func TestMemoryStoreMissingKey(t *testing.T) {
	store := NewMemoryIdempotencyStore()

	_, ok := store.Get("nonexistent")
	if ok {
		t.Error("Get for missing key should return false")
	}
}
