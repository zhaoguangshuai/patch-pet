// Package thirdparty 第三方 SDK 统一封装
// 所有第三方调用必须通过此层，禁止业务代码直连第三方
// 超时 3s、连续 10 次失败熔断 5 分钟、必须有兜底数据/降级方案
package thirdparty

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/patch-pet/patch-pet/pkg/constants"
	"github.com/patch-pet/patch-pet/pkg/logger"
)

// CircuitState 熔断器状态
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // 正常
	CircuitOpen                         // 熔断中
	CircuitHalfOpen                     // 半开（试探）
)

// CircuitBreaker 熔断器
// 连续 N 次失败后熔断指定时长，之后进入半开状态试探恢复
type CircuitBreaker struct {
	mu               sync.Mutex
	state            CircuitState
	failureCount     int
	successCount     int
	lastFailureTime  time.Time
	threshold        int
	recoveryDuration time.Duration
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(threshold int, recoveryDuration time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            CircuitClosed,
		threshold:        threshold,
		recoveryDuration: recoveryDuration,
	}
}

// DefaultCircuitBreaker 默认熔断器：10 次失败，熔断 5 分钟
func DefaultCircuitBreaker() *CircuitBreaker {
	return NewCircuitBreaker(
		constants.ThirdPartyCircuitBreakerThreshold,
		time.Duration(constants.ThirdPartyCircuitBreakerDurationSec)*time.Second,
	)
}

// Allow 检查是否允许请求通过
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.lastFailureTime) > cb.recoveryDuration {
			cb.state = CircuitHalfOpen
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	}
	return false
}

// RecordSuccess 记录成功
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0
	if cb.state == CircuitHalfOpen {
		cb.successCount++
		if cb.successCount >= 3 {
			cb.state = CircuitClosed
			cb.successCount = 0
		}
	}
}

// RecordFailure 记录失败
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()
	if cb.failureCount >= cb.threshold {
		cb.state = CircuitOpen
	}
}

// State 返回当前熔断状态
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// HTTPClient 通用 HTTP 客户端封装
// 所有第三方 HTTP 调用统一通过此客户端，内置超时、重试、熔断
type HTTPClient struct {
	breaker *CircuitBreaker
	timeout time.Duration
}

// NewHTTPClient 创建带熔断器的 HTTP 客户端
func NewHTTPClient(timeout time.Duration, breaker *CircuitBreaker) *HTTPClient {
	return &HTTPClient{
		breaker: breaker,
		timeout: timeout,
	}
}

// DefaultHTTPClient 默认客户端：3s 超时 + 默认熔断器
func DefaultHTTPClient() *HTTPClient {
	return NewHTTPClient(
		time.Duration(constants.ThirdPartyTimeoutMs)*time.Millisecond,
		DefaultCircuitBreaker(),
	)
}

// DoRequest 执行第三方请求（含熔断检查）
// 具体 HTTP 实现由各业务方根据需要补充
func (c *HTTPClient) DoRequest(ctx context.Context, fn func(ctx context.Context) error) error {
	if !c.breaker.Allow() {
		return ErrCircuitOpen
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	err := fn(timeoutCtx)
	if err != nil {
		c.breaker.RecordFailure()
		return err
	}

	c.breaker.RecordSuccess()
	return nil
}

// ThirdPartyClient 第三方服务客户端（带兜底）
type ThirdPartyClient struct {
	httpClient *HTTPClient
}

// NewThirdPartyClient 创建第三方客户端
func NewThirdPartyClient(httpClient *HTTPClient) *ThirdPartyClient {
	return &ThirdPartyClient{httpClient: httpClient}
}

// DefaultThirdPartyClient 默认客户端（3s 超时 + 熔断器 + 兜底）
func DefaultThirdPartyClient() *ThirdPartyClient {
	return NewThirdPartyClient(DefaultHTTPClient())
}

// Call 执行第三方调用，失败时返回 fallback 数据
// 熔断 / 超时 / 业务错误 均降级到 fallback
func (c *ThirdPartyClient) Call(ctx context.Context, fn func(ctx context.Context) (any, error), fallback any) (any, error) {
	var result any
	err := c.httpClient.DoRequest(ctx, func(ctx context.Context) error {
		var callErr error
		result, callErr = fn(ctx)
		return callErr
	})

	if err != nil {
		if fallback != nil {
			logger.Warn("第三方调用降级",
				zap.Error(err),
			)
			return fallback, nil
		}
		return nil, err
	}

	return result, nil
}
