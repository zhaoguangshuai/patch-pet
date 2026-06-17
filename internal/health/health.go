// Package health 健康检查与 K8s 探针
// /health — 存活探针（进程存活）
// /ready — 就绪探针（依赖服务可用）
// 优雅停机最长 30s
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/patch-pet/patch-pet/pkg/constants"
)

// Status 健康状态
type Status string

const (
	StatusUp   Status = "UP"
	StatusDown Status = "DOWN"
)

// CheckResult 单项检查结果
type CheckResult struct {
	Status  Status `json:"status"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]CheckResult `json:"data"`
}

// Checker 健康检查器接口
type Checker interface {
	Name() string
	Check(ctx context.Context) CheckResult
}

// HealthHandler 健康检查 HTTP Handler
type HealthHandler struct {
	checkers []Checker
}

// NewHealthHandler 创建健康检查 Handler
func NewHealthHandler(checkers ...Checker) *HealthHandler {
	return &HealthHandler{checkers: checkers}
}

// Liveness 存活探针 — 仅检查进程是否存活，不检查依赖
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(HealthResponse{
		Code:    0,
		Message: "ok",
		Data:    map[string]CheckResult{"status": {Status: StatusUp}},
	})
}

// Readiness 就绪探针 — 检查所有依赖服务
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	results := make(map[string]CheckResult)
	allUp := true

	for _, c := range h.checkers {
		start := time.Now()
		result := c.Check(ctx)
		result.Latency = time.Since(start).String()
		results[c.Name()] = result
		if result.Status != StatusUp {
			allUp = false
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if allUp {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(HealthResponse{
			Code:    0,
			Message: "ok",
			Data:    results,
		})
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(HealthResponse{
			Code:    5003,
			Message: "服务未就绪",
			Data:    results,
		})
	}
}

// RegisterRoutes 注册健康检查路由
func (h *HealthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.Liveness)
	mux.HandleFunc("/ready", h.Readiness)
}

// --- 常用检查器 ---

// RedisChecker Redis 连接检查
type RedisChecker struct {
	pinger interface {
		Ping(ctx context.Context) error
	}
}

func NewRedisChecker(pinger interface {
	Ping(ctx context.Context) error
}) *RedisChecker {
	return &RedisChecker{pinger: pinger}
}

func (c *RedisChecker) Name() string { return "redis" }

func (c *RedisChecker) Check(ctx context.Context) CheckResult {
	if err := c.pinger.Ping(ctx); err != nil {
		return CheckResult{Status: StatusDown, Message: err.Error()}
	}
	return CheckResult{Status: StatusUp}
}

// DBChecker 数据库连接检查
type DBChecker struct {
	pinger interface {
		PingContext(ctx context.Context) error
	}
}

func NewDBChecker(pinger interface {
	PingContext(ctx context.Context) error
}) *DBChecker {
	return &DBChecker{pinger: pinger}
}

func (c *DBChecker) Name() string { return "postgres" }

func (c *DBChecker) Check(ctx context.Context) CheckResult {
	if err := c.pinger.PingContext(ctx); err != nil {
		return CheckResult{Status: StatusDown, Message: err.Error()}
	}
	return CheckResult{Status: StatusUp}
}

// GracefulShutdown 优雅停机
// 1. 停止接收新请求
// 2. 等待在途请求完成（最长 30s）
// 3. 关闭资源连接
func GracefulShutdown(ctx context.Context, srv *http.Server, closers ...func() error) {
	// 停止 HTTP 服务
	if err := srv.Shutdown(ctx); err != nil {
		// 超时强制关闭
		srv.Close()
	}

	// 关闭资源连接
	for _, closer := range closers {
		closer()
	}
}

// ShutdownTimeout 获取优雅停机超时
func ShutdownTimeout() time.Duration {
	return time.Duration(constants.GracefulShutdownTimeoutSec) * time.Second
}

// EnsureHealthRoutes 确保健康检查路由注册（兼容无 checker 场景）
func EnsureHealthRoutes(mux *http.ServeMux) {
	h := NewHealthHandler()
	h.RegisterRoutes(mux)
}

// EnsureHealthRoutesWithCheckers 带检查器的健康检查路由
func EnsureHealthRoutesWithCheckers(mux *http.ServeMux, checkers ...Checker) {
	h := NewHealthHandler(checkers...)
	h.RegisterRoutes(mux)
}
