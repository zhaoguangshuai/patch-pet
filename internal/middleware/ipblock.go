// Package middleware IP 拦截中间件
// 支持黑名单/白名单模式，配置热加载
package middleware

import (
	"net"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/patch-pet/patch-pet/pkg/constants"
	"github.com/patch-pet/patch-pet/pkg/logger"
)

// IPBlockMode IP 拦截模式
type IPBlockMode string

const (
	IPBlockModeBlacklist IPBlockMode = "blacklist" // 黑名单模式（拦截列表中的 IP）
	IPBlockModeWhitelist IPBlockMode = "whitelist" // 白名单模式（仅放行列表中的 IP）
)

// IPBlockerConfig IP 拦截配置
type IPBlockerConfig struct {
	Mode    IPBlockMode // 拦截模式
	IPList  []string    // IP 列表（支持 CIDR）
	Enabled bool        // 是否启用
}

// IPBlocker IP 拦截器
type IPBlocker struct {
	mu       sync.RWMutex
	config   IPBlockerConfig
	ipNets   []*net.IPNet
}

// NewIPBlocker 创建 IP 拦截器
func NewIPBlocker(cfg IPBlockerConfig) *IPBlocker {
	blocker := &IPBlocker{
		config: cfg,
	}
	if cfg.Enabled {
		blocker.parseIPList()
	}
	return blocker
}

func (b *IPBlocker) parseIPList() {
	b.ipNets = nil
	for _, ip := range b.config.IPList {
		// 尝试解析为 CIDR
		if _, ipNet, err := net.ParseCIDR(ip); err == nil {
			b.ipNets = append(b.ipNets, ipNet)
			continue
		}
		// 尝试解析为单个 IP
		if parsed := net.ParseIP(ip); parsed != nil {
			mask := net.CIDRMask(32, 32)
			if parsed.To4() == nil {
				mask = net.CIDRMask(128, 128)
			}
			b.ipNets = append(b.ipNets, &net.IPNet{IP: parsed, Mask: mask})
		}
	}
}

// UpdateConfig 热更新配置
func (b *IPBlocker) UpdateConfig(cfg IPBlockerConfig) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.config = cfg
	if cfg.Enabled {
		b.parseIPList()
	}
}

// IsBlocked 检查 IP 是否被拦截
func (b *IPBlocker) IsBlocked(ipStr string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.config.Enabled {
		return false
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	matched := false
	for _, ipNet := range b.ipNets {
		if ipNet.Contains(ip) {
			matched = true
			break
		}
	}

	switch b.config.Mode {
	case IPBlockModeBlacklist:
		return matched // 命中黑名单 → 拦截
	case IPBlockModeWhitelist:
		return !matched // 未命中白名单 → 拦截
	default:
		return false
	}
}

// Middleware 返回 Gin IP 拦截中间件
func (b *IPBlocker) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		if b.IsBlocked(clientIP) {
			logger.Warn("IP 被拦截",
				zap.String("ip", clientIP),
				zap.String("mode", string(b.config.Mode)),
				zap.String("path", c.Request.URL.Path),
			)
			c.JSON(http.StatusForbidden, gin.H{
				"code":    constants.CodeAuthForbidden,
				"message": "访问被拒绝",
				"data":    nil,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
