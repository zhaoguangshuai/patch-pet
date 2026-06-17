// Package middleware API 网关限流中间件
// 基于 Redis 的滑动窗口限流，支持 IP 级别和接口级别
package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/patch-pet/patch-pet/pkg/constants"
	"github.com/patch-pet/patch-pet/pkg/logger"
	"github.com/patch-pet/patch-pet/pkg/thirdparty"
)

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	// MaxRequests 窗口内最大请求数
	MaxRequests int
	// Window 时间窗口
	Window time.Duration
	// KeyFunc 限流键生成函数（默认按 IP）
	KeyFunc func(c *gin.Context) string
}

// DefaultRateLimitConfig 默认限流配置
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		MaxRequests: 100,
		Window:      time.Minute,
		KeyFunc: func(c *gin.Context) string {
			return c.ClientIP()
		},
	}
}

// RateLimiter 基于 Redis 的滑动窗口限流器
type RateLimiter struct {
	redis  *thirdparty.RedisClient
	config RateLimitConfig
}

// NewRateLimiter 创建限流器
func NewRateLimiter(redis *thirdparty.RedisClient, cfg RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		redis:  redis,
		config: cfg,
	}
}

// Middleware 返回 Gin 限流中间件
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("ratelimit:%s", rl.config.KeyFunc(c))

		// 使用 Redis INCR + EXPIRE 实现简单限流
		count, err := rl.redis.Incr(c.Request.Context(), key)
		if err != nil {
			// Redis 故障时放行（宁可放过不可误杀）
			logger.Error("限流器 Redis 操作失败", zap.Error(err))
			c.Next()
			return
		}

		// 首次请求设置过期时间
		if count == 1 {
			_ = rl.redis.Expire(c.Request.Context(), key, rl.config.Window)
		}

		if count > int64(rl.config.MaxRequests) {
			logger.Warn("请求被限流",
				zap.String("key", key),
				zap.Int64("count", count),
				zap.Int("max", rl.config.MaxRequests),
			)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    constants.CodeAuthForbidden,
				"message": "请求过于频繁，请稍后再试",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 设置限流响应头
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.config.MaxRequests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", rl.config.MaxRequests-int(count)))

		c.Next()
	}
}
