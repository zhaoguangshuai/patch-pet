package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockPinger 模拟 Redis/DB ping
type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(ctx context.Context) error        { return m.err }
func (m *mockPinger) PingContext(ctx context.Context) error  { return m.err }

func TestLivenessAlwaysOK(t *testing.T) {
	h := NewHealthHandler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.Liveness(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp HealthResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 0 {
		t.Errorf("code = %d, want 0", resp.Code)
	}
}

func TestReadinessNoCheckers(t *testing.T) {
	h := NewHealthHandler()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	h.Readiness(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestReadinessAllUp(t *testing.T) {
	h := NewHealthHandler(
		NewRedisChecker(&mockPinger{err: nil}),
		NewDBChecker(&mockPinger{err: nil}),
	)
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	h.Readiness(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp HealthResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data["redis"].Status != StatusUp {
		t.Error("redis should be UP")
	}
	if resp.Data["postgres"].Status != StatusUp {
		t.Error("postgres should be UP")
	}
}

func TestReadinessOneDown(t *testing.T) {
	h := NewHealthHandler(
		NewRedisChecker(&mockPinger{err: nil}),
		NewDBChecker(&mockPinger{err: errors.New("connection refused")}),
	)
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	h.Readiness(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var resp HealthResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 5003 {
		t.Errorf("code = %d, want 5003", resp.Code)
	}
	if resp.Data["postgres"].Status != StatusDown {
		t.Error("postgres should be DOWN")
	}
}

func TestRedisCheckerName(t *testing.T) {
	c := NewRedisChecker(&mockPinger{})
	if c.Name() != "redis" {
		t.Errorf("name = %s, want redis", c.Name())
	}
}

func TestDBCheckerName(t *testing.T) {
	c := NewDBChecker(&mockPinger{})
	if c.Name() != "postgres" {
		t.Errorf("name = %s, want postgres", c.Name())
	}
}

func TestShutdownTimeout(t *testing.T) {
	d := ShutdownTimeout()
	if d != 30*1e9 { // 30s in nanoseconds
		t.Errorf("timeout = %v, want 30s", d)
	}
}
