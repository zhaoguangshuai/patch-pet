// Package middleware API 网关
// 组装限流、IP 拦截、链路追踪等中间件（Gin 原生）
package middleware

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/patch-pet/patch-pet/internal/auth"
	"github.com/patch-pet/patch-pet/pkg/constants"
	"github.com/patch-pet/patch-pet/pkg/logger"
	"github.com/patch-pet/patch-pet/pkg/thirdparty"
	"github.com/patch-pet/patch-pet/pkg/types"
	"github.com/patch-pet/patch-pet/pkg/utils"
)

// GatewayConfig 网关配置
type GatewayConfig struct {
	RateLimit   RateLimitConfig
	IPBlocker   IPBlockerConfig
	EnableTrace bool
}

// DefaultGatewayConfig 默认网关配置
func DefaultGatewayConfig() GatewayConfig {
	return GatewayConfig{
		RateLimit:   DefaultRateLimitConfig(),
		IPBlocker:   IPBlockerConfig{Enabled: false},
		EnableTrace: true,
	}
}

// SetupGateway 组装 API 网关中间件链
// 执行顺序：IP 拦截 → 限流 → 链路追踪 → Header 校验 → 认证 → 分页
func SetupGateway(r *gin.Engine, redis *thirdparty.RedisClient, cfg GatewayConfig) {
	// 1. IP 拦截
	if cfg.IPBlocker.Enabled {
		blocker := NewIPBlocker(cfg.IPBlocker)
		r.Use(blocker.Middleware())
	}

	// 2. 限流
	if redis != nil {
		limiter := NewRateLimiter(redis, cfg.RateLimit)
		r.Use(limiter.Middleware())
	}

	// 3. 链路追踪
	if cfg.EnableTrace {
		r.Use(GinTraceMiddleware())
	}

	// 4. Header 校验
	r.Use(GinHeaderValidation())

	// 5. 分页
	r.Use(GinPaginationMiddleware())
}

// GinTraceMiddleware Gin 原生链路追踪中间件
func GinTraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Trace-Id")

		if traceID == "" || !isValidULIDFormat(traceID) {
			traceID = utils.GenerateULID(types.IDPrefix("trace"))
		}

		ctx := context.WithValue(c.Request.Context(), TraceIDKey, traceID)
		c.Request = c.Request.WithContext(ctx)

		c.Header("X-Trace-Id", traceID)

		traceLogger := logger.WithTraceID(traceID)
		traceLogger.Debug("请求开始",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("ip", c.ClientIP()),
		)

		c.Next()
	}
}

// GinHeaderValidation Gin 原生 Header 校验中间件
func GinHeaderValidation() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 健康检查跳过
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    constants.CodeAuthHeaderMissing,
				"message": "缺少 Authorization Header",
				"data":    nil,
			})
			c.Abort()
			return
		}

		source := c.GetHeader("X-Request-Source")
		if source == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    constants.CodeAuthHeaderMissing,
				"message": "缺少 X-Request-Source Header",
				"data":    nil,
			})
			c.Abort()
			return
		}

		ctx := context.WithValue(c.Request.Context(), RequestSourceKey, source)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// GinPaginationMiddleware Gin 原生分页中间件
func GinPaginationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		pageNum := parseIntQuery(c, "pageNum", 1)
		pageSize := parseIntQuery(c, "pageSize", constants.DefaultPageSize)

		if pageNum < 1 {
			pageNum = 1
		}
		if pageSize > constants.MaxPageSize {
			pageSize = constants.MaxPageSize
		}
		if pageSize < 1 {
			pageSize = constants.DefaultPageSize
		}

		c.Set("pageNum", pageNum)
		c.Set("pageSize", pageSize)

		c.Next()
	}
}

func parseIntQuery(c *gin.Context, key string, defaultVal int) int {
	s := c.Query(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

// GinAuthMiddleware Gin 认证中间件
// 从 Authorization Header 提取 Token，解析用户身份注入 context
func GinAuthMiddleware(authenticator auth.Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 健康检查跳过
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    constants.CodeAuthHeaderMissing,
				"message": "缺少 Authorization Header",
				"data":    nil,
			})
			c.Abort()
			return
		}

		reqCtx, err := authenticator.Authenticate(c.Request.Context(), token)
		if err != nil {
			logger.Warn("认证失败",
				zap.String("path", c.Request.URL.Path),
				zap.Error(err),
			)
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    constants.CodeAuthUnauthorized,
				"message": "认证失败: " + err.Error(),
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 注入认证上下文
		ctx := context.WithValue(c.Request.Context(), RequestContextKey, reqCtx)
		ctx = context.WithValue(ctx, UserIDKey, reqCtx.UserID)
		c.Request = c.Request.WithContext(ctx)
		c.Set("requestContext", reqCtx)

		c.Next()
	}
}

// GetRequestContext 从 Gin context 获取认证上下文
func GetRequestContext(c *gin.Context) *auth.RequestContext {
	if v, exists := c.Get("requestContext"); exists {
		if rc, ok := v.(*auth.RequestContext); ok {
			return rc
		}
	}
	return nil
}
