// Package middleware HTTP 中间件
// 全链路 Trace（X-Trace-Id）中间件：请求入口生成或透传 ULID 格式 trace_id
// 全局请求 Header 校验：Authorization、X-Trace-Id、X-Request-Source
package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/patch-pet/patch-pet/pkg/utils"
	"github.com/patch-pet/patch-pet/pkg/types"
	"github.com/patch-pet/patch-pet/pkg/logger"

	"go.uber.org/zap"
)

// ContextKey context 键类型
type ContextKey string

const (
	// TraceIDKey trace_id 在 context 中的键
	TraceIDKey ContextKey = "trace_id"
	// UserIDKey 用户 ID 在 context 中的键
	UserIDKey ContextKey = "user_id"
	// RequestSourceKey 请求来源在 context 中的键
	RequestSourceKey ContextKey = "request_source"
	// RequestContextKey 请求上下文（认证结果）在 context 中的键
	RequestContextKey ContextKey = "request_context"
)

// TraceMiddleware 全链路追踪中间件
// 1. 优先从 X-Trace-Id Header 透传
// 2. 不存在时自动生成 ULID 格式 trace_id
// 3. 注入 context 供下游使用
func TraceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("X-Trace-Id")

		// 校验或生成 trace_id
		if traceID == "" || !isValidULIDFormat(traceID) {
			traceID = utils.GenerateULID(types.IDPrefix("trace"))
		}

		// 注入 context
		ctx := context.WithValue(r.Context(), TraceIDKey, traceID)
		r = r.WithContext(ctx)

		// 设置响应 Header
		w.Header().Set("X-Trace-Id", traceID)

		// 创建带 trace_id 的 logger
		traceLogger := logger.WithTraceID(traceID)
		traceLogger.Debug("请求开始",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
		)

		next.ServeHTTP(w, r)
	})
}

// HeaderValidationMiddleware 全局请求 Header 校验
// 必须携带：Authorization、X-Trace-Id、X-Request-Source
func HeaderValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 健康检查接口跳过校验
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		// 校验 Authorization
		auth := r.Header.Get("Authorization")
		if auth == "" {
			writeErrorResponse(w, http.StatusUnauthorized, 1003, "缺少 Authorization Header")
			return
		}

		// 校验 X-Request-Source
		source := r.Header.Get("X-Request-Source")
		if source == "" {
			writeErrorResponse(w, http.StatusBadRequest, 1003, "缺少 X-Request-Source Header")
			return
		}

		// 注入请求来源到 context
		ctx := context.WithValue(r.Context(), RequestSourceKey, source)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// GetTraceID 从 context 获取 trace_id
func GetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(TraceIDKey).(string); ok {
		return v
	}
	return ""
}

// GetUserID 从 context 获取 user_id
func GetUserID(ctx context.Context) string {
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		return v
	}
	return ""
}

// GetRequestSource 从 context 获取请求来源
func GetRequestSource(ctx context.Context) string {
	if v, ok := ctx.Value(RequestSourceKey).(string); ok {
		return v
	}
	return ""
}

// isValidULIDFormat 校验 ULID 格式（26 字符 Crockford Base32）
// 支持带前缀的格式：trace_01JZYQ4M8PABCDEFGHJKLMNP（32 字符）
// 或纯 ULID 格式：01JZYQ4M8PABCDEFGHJKLMNP（26 字符）
func isValidULIDFormat(s string) bool {
	// 带前缀格式：trace_ + 26 字符
	if len(s) == 32 && s[:6] == "trace_" {
		return isValidCrockfordBase32(s[6:])
	}
	// 纯 ULID 格式：26 字符
	if len(s) == 26 {
		return isValidCrockfordBase32(s)
	}
	return false
}

// isValidCrockfordBase32 校验 Crockford Base32 字符串
func isValidCrockfordBase32(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'H') ||
			(c >= 'J' && c <= 'K') || (c >= 'M' && c <= 'N') ||
			(c >= 'P' && c <= 'T') || (c >= 'V' && c <= 'Z') ||
			(c >= 'a' && c <= 'h') || (c >= 'j' && c <= 'k') ||
			(c >= 'm' && c <= 'n') || (c >= 'p' && c <= 't') ||
			(c >= 'v' && c <= 'z')) {
			return false
		}
	}
	return true
}

// writeErrorResponse 写入错误响应
func writeErrorResponse(w http.ResponseWriter, statusCode int, bizCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	// 使用统一响应格式
	response := fmt.Sprintf(`{"code":%d,"message":"%s","data":null}`, bizCode, message)
	w.Write([]byte(response))
}
