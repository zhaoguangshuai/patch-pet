// Package utils 通用重试工具
// 普通接口重试 2 次 / LLM 重试 1 次 / MQ 按队列配置
package utils

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/patch-pet/patch-pet/pkg/logger"
)

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries  int           // 最大重试次数（不含首次调用）
	Delay       time.Duration // 重试间隔
	MaxDelay    time.Duration // 最大退避延迟
	IsRetryable func(error) bool // 判断是否可重试
}

// RetryProfile 预设重试策略
var (
	// NormalAPIRetry 普通接口：2 次重试，200ms 间隔
	NormalAPIRetry = RetryConfig{
		MaxRetries: 2,
		Delay:      200 * time.Millisecond,
		MaxDelay:   1 * time.Second,
		IsRetryable: func(err error) bool {
			return err != nil
		},
	}

	// LLMRetry LLM 调用：1 次重试，500ms 间隔
	LLMRetry = RetryConfig{
		MaxRetries: 1,
		Delay:      500 * time.Millisecond,
		MaxDelay:   2 * time.Second,
		IsRetryable: func(err error) bool {
			return err != nil
		},
	}

	// MQRetryDefault MQ 默认：3 次重试，1s 间隔
	MQRetryDefault = RetryConfig{
		MaxRetries: 3,
		Delay:      1 * time.Second,
		MaxDelay:   10 * time.Second,
		IsRetryable: func(err error) bool {
			return err != nil
		},
	}

	// MQRetryDLQ MQ 死信：0 次重试（直接进 DLQ）
	MQRetryDLQ = RetryConfig{
		MaxRetries: 0,
		Delay:      0,
		MaxDelay:   0,
		IsRetryable: func(err error) bool {
			return false
		},
	}
)

// RetryableError 可重试错误包装
type RetryableError struct {
	Err       error
	Retryable bool
}

func (e *RetryableError) Error() string { return e.Err.Error() }
func (e *RetryableError) Unwrap() error { return e.Err }

// NewRetryableError 创建可重试错误
func NewRetryableError(err error, retryable bool) *RetryableError {
	return &RetryableError{Err: err, Retryable: retryable}
}

// Retry 执行带重试的操作
func Retry(ctx context.Context, cfg RetryConfig, fn func(ctx context.Context) error) error {
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := cfg.Delay * time.Duration(1<<(attempt-1))
			if cfg.MaxDelay > 0 && delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}

			select {
			case <-ctx.Done():
				return fmt.Errorf("重试取消: %w", ctx.Err())
			case <-time.After(delay):
			}

			logger.Warn("重试操作",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", cfg.MaxRetries),
				zap.Error(lastErr),
			)
		}

		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// 检查是否可重试
		if re, ok := err.(*RetryableError); ok {
			if !re.Retryable {
				return err
			}
		} else if cfg.IsRetryable != nil && !cfg.IsRetryable(err) {
			return err
		}
	}

	return fmt.Errorf("重试 %d 次后仍失败: %w", cfg.MaxRetries, lastErr)
}

// RetryWithResult 带返回值的重试
func RetryWithResult[T any](ctx context.Context, cfg RetryConfig, fn func(ctx context.Context) (T, error)) (T, error) {
	var zero T
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := cfg.Delay * time.Duration(1<<(attempt-1))
			if cfg.MaxDelay > 0 && delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}

			select {
			case <-ctx.Done():
				return zero, fmt.Errorf("重试取消: %w", ctx.Err())
			case <-time.After(delay):
			}
		}

		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}

		lastErr = err

		if re, ok := err.(*RetryableError); ok {
			if !re.Retryable {
				return zero, err
			}
		} else if cfg.IsRetryable != nil && !cfg.IsRetryable(err) {
			return zero, err
		}
	}

	return zero, fmt.Errorf("重试 %d 次后仍失败: %w", cfg.MaxRetries, lastErr)
}
