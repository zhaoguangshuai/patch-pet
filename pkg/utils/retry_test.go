package utils

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetrySuccess(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries: 2,
		Delay:      10 * time.Millisecond,
		IsRetryable: func(err error) bool { return true },
	}

	attempts := 0
	err := Retry(context.Background(), cfg, func(ctx context.Context) error {
		attempts++
		if attempts < 2 {
			return errors.New("fail")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}

func TestRetryExhausted(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries: 2,
		Delay:      10 * time.Millisecond,
		IsRetryable: func(err error) bool { return true },
	}

	attempts := 0
	err := Retry(context.Background(), cfg, func(ctx context.Context) error {
		attempts++
		return errors.New("always fail")
	})

	if err == nil {
		t.Error("expected error after retries exhausted")
	}
	if attempts != 3 { // 1 initial + 2 retries
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetryNotRetryable(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries: 3,
		Delay:      10 * time.Millisecond,
		IsRetryable: func(err error) bool { return true },
	}

	attempts := 0
	err := Retry(context.Background(), cfg, func(ctx context.Context) error {
		attempts++
		return NewRetryableError(errors.New("fatal"), false)
	})

	if err == nil {
		t.Error("expected error")
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (non-retryable)", attempts)
	}
}

func TestRetryCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	cfg := RetryConfig{
		MaxRetries: 3,
		Delay:      1 * time.Second,
		IsRetryable: func(err error) bool { return true },
	}

	err := Retry(ctx, cfg, func(ctx context.Context) error {
		return errors.New("fail")
	})
	if err == nil {
		t.Error("expected cancel error")
	}
}

func TestRetryZeroRetries(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries: 0,
		Delay:      10 * time.Millisecond,
		IsRetryable: func(err error) bool { return true },
	}

	attempts := 0
	err := Retry(context.Background(), cfg, func(ctx context.Context) error {
		attempts++
		return errors.New("fail")
	})

	if err == nil {
		t.Error("expected error")
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestRetryWithResultSuccess(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries: 2,
		Delay:      10 * time.Millisecond,
		IsRetryable: func(err error) bool { return true },
	}

	result, err := RetryWithResult(context.Background(), cfg, func(ctx context.Context) (string, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ok" {
		t.Errorf("result = %s, want ok", result)
	}
}

func TestRetryWithResultFallback(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries: 2,
		Delay:      10 * time.Millisecond,
		IsRetryable: func(err error) bool { return true },
	}

	attempts := 0
	result, err := RetryWithResult(context.Background(), cfg, func(ctx context.Context) (int, error) {
		attempts++
		if attempts < 3 {
			return 0, errors.New("fail")
		}
		return 42, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("result = %d, want 42", result)
	}
}

func TestNormalAPIRetryConfig(t *testing.T) {
	if NormalAPIRetry.MaxRetries != 2 {
		t.Errorf("NormalAPIRetry.MaxRetries = %d, want 2", NormalAPIRetry.MaxRetries)
	}
}

func TestLLMRetryConfig(t *testing.T) {
	if LLMRetry.MaxRetries != 1 {
		t.Errorf("LLMRetry.MaxRetries = %d, want 1", LLMRetry.MaxRetries)
	}
}

func TestMQRetryDefaultConfig(t *testing.T) {
	if MQRetryDefault.MaxRetries != 3 {
		t.Errorf("MQRetryDefault.MaxRetries = %d, want 3", MQRetryDefault.MaxRetries)
	}
}

func TestMQRetryDLQConfig(t *testing.T) {
	if MQRetryDLQ.MaxRetries != 0 {
		t.Errorf("MQRetryDLQ.MaxRetries = %d, want 0", MQRetryDLQ.MaxRetries)
	}
}
