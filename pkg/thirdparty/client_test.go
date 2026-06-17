package thirdparty

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerClosed(t *testing.T) {
	cb := NewCircuitBreaker(3, 1*time.Second)
	if !cb.Allow() {
		t.Error("new breaker should allow")
	}
	if cb.State() != CircuitClosed {
		t.Error("new breaker should be closed")
	}
}

func TestCircuitBreakerOpens(t *testing.T) {
	cb := NewCircuitBreaker(3, 1*time.Second)
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}
	if cb.Allow() {
		t.Error("breaker should be open after 3 failures")
	}
	if cb.State() != CircuitOpen {
		t.Error("breaker state should be open")
	}
}

func TestCircuitBreakerRecovers(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond)
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.Allow() {
		t.Error("breaker should be open")
	}
	time.Sleep(60 * time.Millisecond)
	if !cb.Allow() {
		t.Error("breaker should be half-open after recovery")
	}
	// 3 successes to close
	cb.RecordSuccess()
	cb.RecordSuccess()
	cb.RecordSuccess()
	if cb.State() != CircuitClosed {
		t.Error("breaker should be closed after 3 successes")
	}
}

func TestCircuitBreakerResetOnSuccess(t *testing.T) {
	cb := NewCircuitBreaker(3, 1*time.Second)
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess()
	cb.RecordFailure()
	cb.RecordFailure()
	// Only 2 failures after success, should still be closed
	if !cb.Allow() {
		t.Error("breaker should still be closed")
	}
}

func TestHTTPClientDoRequestSuccess(t *testing.T) {
	c := NewHTTPClient(1*time.Second, NewCircuitBreaker(3, 1*time.Second))
	err := c.DoRequest(context.Background(), func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestHTTPClientDoRequestFailure(t *testing.T) {
	c := NewHTTPClient(1*time.Second, NewCircuitBreaker(3, 1*time.Second))
	testErr := errors.New("service error")
	err := c.DoRequest(context.Background(), func(ctx context.Context) error {
		return testErr
	})
	if err != testErr {
		t.Errorf("expected testErr, got %v", err)
	}
}

func TestHTTPClientDoRequestCircuitOpen(t *testing.T) {
	cb := NewCircuitBreaker(2, 1*time.Second)
	c := NewHTTPClient(1*time.Second, cb)

	// Trip the breaker
	for i := 0; i < 2; i++ {
		c.DoRequest(context.Background(), func(ctx context.Context) error {
			return errors.New("fail")
		})
	}

	err := c.DoRequest(context.Background(), func(ctx context.Context) error {
		return nil
	})
	if err != ErrCircuitOpen {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestHTTPClientDoRequestTimeout(t *testing.T) {
	c := NewHTTPClient(50*time.Millisecond, NewCircuitBreaker(3, 1*time.Second))
	err := c.DoRequest(context.Background(), func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return nil
		}
	})
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestThirdPartyClientCallSuccess(t *testing.T) {
	c := DefaultThirdPartyClient()
	result, err := c.Call(context.Background(), func(ctx context.Context) (any, error) {
		return "ok", nil
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ok" {
		t.Errorf("result = %v, want ok", result)
	}
}

func TestThirdPartyClientCallWithFallback(t *testing.T) {
	c := DefaultThirdPartyClient()
	result, err := c.Call(context.Background(), func(ctx context.Context) (any, error) {
		return nil, errors.New("service down")
	}, "fallback_value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "fallback_value" {
		t.Errorf("result = %v, want fallback_value", result)
	}
}

func TestThirdPartyClientCallNoFallback(t *testing.T) {
	c := DefaultThirdPartyClient()
	_, err := c.Call(context.Background(), func(ctx context.Context) (any, error) {
		return nil, errors.New("service down")
	}, nil)
	if err == nil {
		t.Error("expected error when no fallback")
	}
}

func TestDefaultHTTPClientHasCorrectTimeout(t *testing.T) {
	c := DefaultHTTPClient()
	if c.timeout != 3*time.Second {
		t.Errorf("timeout = %v, want 3s", c.timeout)
	}
}
