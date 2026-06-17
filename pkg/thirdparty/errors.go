package thirdparty

import "errors"

// ErrCircuitOpen 熔断器打开，拒绝请求
var ErrCircuitOpen = errors.New("circuit breaker is open, request rejected")

// ErrTimeout 第三方请求超时
var ErrTimeout = errors.New("third-party request timeout")

// ErrDegraded 降级至兜底方案
var ErrDegraded = errors.New("degraded to fallback")
