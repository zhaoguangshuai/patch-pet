// Package thirdparty 第三方 SDK 统一封装
// Unleash 功能开关客户端：灰度/特性开关统一收口
// 新增工具/权限/能力默认禁用，需人工审批后方可上线
// Unleash 不可用时所有 feature flag 默认返回 false（Default-Deny）
package thirdparty

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/Unleash/unleash-client-go/v4"
	"github.com/Unleash/unleash-client-go/v4/api"
	"github.com/patch-pet/patch-pet/pkg/logger"
	"go.uber.org/zap"
)

// UnleashConfig Unleash 连接配置
// 实际值从环境变量注入，禁止硬编码
type UnleashConfig struct {
	URL        string // Unleash Server 地址（环境变量 UNLEASH_URL）
	AppName    string // 应用名称（环境变量 UNLEASH_APP_NAME）
	InstanceID string // 实例标识（环境变量 UNLEASH_INSTANCE_ID）
}

// featureFlagListener Unleash 事件监听器
// 实现 OnReady/OnError 接口，用于感知连接状态
type featureFlagListener struct {
	ready *atomic.Bool
}

func (l *featureFlagListener) OnReady() {
	l.ready.Store(true)
	logger.Info("Unleash 客户端就绪")
}

func (l *featureFlagListener) OnError(err error) {
	logger.Warn("Unleash 客户端错误", zap.Error(err))
}

// FeatureFlagClient 功能开关客户端
// 封装 Unleash SDK，提供 Default-Deny 降级策略
type FeatureFlagClient struct {
	config   UnleashConfig
	listener *featureFlagListener
}

// NewFeatureFlagClient 创建功能开关客户端
// 配置从环境变量注入，禁止硬编码地址/密钥
func NewFeatureFlagClient(cfg UnleashConfig) *FeatureFlagClient {
	return &FeatureFlagClient{
		config: cfg,
		listener: &featureFlagListener{
			ready: &atomic.Bool{},
		},
	}
}

// Init 初始化客户端，连接 Unleash Server
// 连接失败时降级为默认配置（所有 flag 返回 false），不阻塞服务启动
func (c *FeatureFlagClient) Init() error {
	if c.config.URL == "" {
		c.config.URL = defaultEnv("UNLEASH_URL", "http://localhost:4242")
	}
	if c.config.AppName == "" {
		c.config.AppName = defaultEnv("UNLEASH_APP_NAME", "patch-pet")
	}
	if c.config.InstanceID == "" {
		c.config.InstanceID = defaultEnv("UNLEASH_INSTANCE_ID", fmt.Sprintf("patch-pet-%d", time.Now().UnixMilli()))
	}

	err := unleash.Initialize(
		unleash.WithUrl(c.config.URL),
		unleash.WithAppName(c.config.AppName),
		unleash.WithInstanceId(c.config.InstanceID),
		unleash.WithRefreshInterval(10*time.Second),
		unleash.WithMetricsInterval(30*time.Second),
		unleash.WithListener(c.listener),
	)
	if err != nil {
		// Unleash 不可用时降级为默认配置，不阻塞启动
		logger.Warn("Unleash 初始化失败，功能开关默认关闭",
			zap.Error(err),
			zap.String("url", c.config.URL),
		)
		return nil
	}

	return nil
}

// IsEnabled 检查功能开关是否启用
// Unleash 未就绪或连接失败时默认返回 false（Default-Deny）
func (c *FeatureFlagClient) IsEnabled(flagName string) bool {
	if !c.listener.ready.Load() {
		return false
	}
	return unleash.IsEnabled(flagName)
}

// IsEnabledWithDefault 检查功能开关（含默认值）
// 仅在需要显式默认值时使用，常规场景使用 IsEnabled
func (c *FeatureFlagClient) IsEnabledWithDefault(flagName string, defaultVal bool) bool {
	if !c.listener.ready.Load() {
		return defaultVal
	}
	return unleash.IsEnabled(flagName, unleash.WithFallback(defaultVal))
}

// GetVariant 获取功能开关变体（A/B 测试、灰度比例控制）
func (c *FeatureFlagClient) GetVariant(flagName string) *api.Variant {
	if !c.listener.ready.Load() {
		return &api.Variant{
			Name:    "disabled",
			Enabled: false,
		}
	}
	return unleash.GetVariant(flagName)
}

// Ready 客户端是否就绪
func (c *FeatureFlagClient) Ready() bool {
	return c.listener.ready.Load()
}

// defaultEnv 从环境变量读取值，不存在时返回默认值
func defaultEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
