// Package thirdparty 第三方 SDK 统一封装
// Nacos 配置中心客户端
// 所有超时/限流/地址/密钥/阈值统一托管，代码禁止硬编码
package thirdparty

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// NacosConfig Nacos 连接配置
// 实际值从环境变量注入，禁止硬编码
type NacosConfig struct {
	Addr      string // Nacos 地址（环境变量 NACOS_ADDR）
	Port      uint64 // Nacos 端口（环境变量 NACOS_PORT）
	Namespace string // 命名空间（环境变量 NACOS_NAMESPACE）
	Group     string // 配置分组
	DataID    string // 配置 DataID
	Username  string // 用户名（环境变量 NACOS_USERNAME）
	Password  string // 密码（环境变量 NACOS_PASSWORD）
}

// AppConfig 应用配置（从 Nacos 加载）
// 所有配置项禁止在代码中硬编码，统一走 Nacos
type AppConfig struct {
	// 全局运行阈值
	MaxToolCallsPerSession    int `json:"max_tool_calls_per_session"`
	MaxToolNestingDepth       int `json:"max_tool_nesting_depth"`
	MaxLLMInputTokens         int `json:"max_llm_input_tokens"`
	MaxLLMOutputTokens        int `json:"max_llm_output_tokens"`
	ThirdPartyTimeoutMs       int `json:"third_party_timeout_ms"`
	CircuitBreakerThreshold   int `json:"circuit_breaker_threshold"`
	CircuitBreakerDurationSec int `json:"circuit_breaker_duration_sec"`
	ScheduledTaskTimeoutSec   int `json:"scheduled_task_timeout_sec"`
	GracefulShutdownSec       int `json:"graceful_shutdown_sec"`

	// 分页
	DefaultPageSize int `json:"default_page_size"`
	MaxPageSize     int `json:"max_page_size"`

	// 防重放
	ReplayToleranceSec int `json:"replay_tolerance_sec"`

	// Token 预算
	DailyTokenBudget   int `json:"daily_token_budget"`
	MonthlyTokenBudget int `json:"monthly_token_budget"`
	MaxCostPerUser     int `json:"max_cost_per_user"`

	// Kafka
	KafkaBrokers   string `json:"kafka_brokers"`
	KafkaMaxRetries int   `json:"kafka_max_retries"`

	// Redis
	RedisAddr         string `json:"redis_addr"`
	RedisPassword     string `json:"redis_password"`
	SessionTTLMiuntes int   `json:"session_ttl_minutes"`

	// PostgreSQL
	PostgresDSN string `json:"postgres_dsn"`

	// Unleash 功能开关
	UnleashURL       string `json:"unleash_url"`
	UnleashAppName   string `json:"unleash_app_name"`
	UnleashInstanceID string `json:"unleash_instance_id"`
}

// DefaultConfig 返回默认配置（Nacos 不可用时的兜底）
func DefaultConfig() *AppConfig {
	return &AppConfig{
		MaxToolCallsPerSession:    5,
		MaxToolNestingDepth:       3,
		MaxLLMInputTokens:         8000,
		MaxLLMOutputTokens:        1500,
		ThirdPartyTimeoutMs:       3000,
		CircuitBreakerThreshold:   10,
		CircuitBreakerDurationSec: 300,
		ScheduledTaskTimeoutSec:   10,
		GracefulShutdownSec:       30,
		DefaultPageSize:           20,
		MaxPageSize:               50,
		ReplayToleranceSec:        300,
		DailyTokenBudget:          100000,
		MonthlyTokenBudget:        2000000,
		MaxCostPerUser:            5000,
		KafkaMaxRetries:           3,
		SessionTTLMiuntes:         1440, // 24h
	}
}

// ConfigChangeListener 配置变更监听器
type ConfigChangeListener func(newConfig *AppConfig)

// NacosClient Nacos 配置中心客户端
// 封装配置加载、变更监听、热更新
type NacosClient struct {
	mu              sync.RWMutex
	config          *AppConfig
	nacosCfg        NacosConfig
	listeners       []ConfigChangeListener
	watchCancel     context.CancelFunc
}

// NewNacosClient 创建 Nacos 客户端
// 配置从环境变量注入，禁止硬编码地址/密钥
func NewNacosClient(cfg NacosConfig) *NacosClient {
	return &NacosClient{
		config:   DefaultConfig(),
		nacosCfg: cfg,
	}
}

// Init 初始化客户端，加载配置并启动监听
func (c *NacosClient) Init(ctx context.Context) error {
	// 首次加载配置
	if err := c.loadConfig(ctx); err != nil {
		// Nacos 不可用时使用默认配置，不阻塞启动
		fmt.Printf("[WARN] Nacos 配置加载失败，使用默认配置: %v\n", err)
		return nil
	}

	// 启动配置变更监听
	c.startWatch(ctx)
	return nil
}

// GetConfig 获取当前配置（线程安全）
func (c *NacosClient) GetConfig() *AppConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// OnConfigChange 注册配置变更监听器
func (c *NacosClient) OnConfigChange(listener ConfigChangeListener) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.listeners = append(c.listeners, listener)
}

// loadConfig 从 Nacos 加载配置
func (c *NacosClient) loadConfig(ctx context.Context) error {
	// ASSUMPTION: 具体的 Nacos SDK 调用逻辑在集成 nacos-sdk-go 时实现
	// 此处定义接口契约，确保配置结构正确
	//
	// 集成时实现：
	// 1. 创建 Nacos naming/config client
	// 2. 调用 configClient.GetConfig() 获取配置 JSON
	// 3. 反序列化为 AppConfig
	rawConfig, err := c.fetchFromNacos(ctx)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.config = rawConfig
	c.mu.Unlock()

	return nil
}

// fetchFromNacos 从 Nacos 获取原始配置
func (c *NacosClient) fetchFromNacos(ctx context.Context) (*AppConfig, error) {
	// ASSUMPTION: 具体的 HTTP/gRPC 调用待集成 nacos-sdk-go
	// 此处返回错误表示需要集成
	return nil, fmt.Errorf("Nacos SDK 待集成，当前使用默认配置")
}

// startWatch 启动配置变更监听
func (c *NacosClient) startWatch(ctx context.Context) {
	watchCtx, cancel := context.WithCancel(ctx)
	c.watchCancel = cancel

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-watchCtx.Done():
				return
			case <-ticker.C:
				c.checkAndUpdate(watchCtx)
			}
		}
	}()
}

// checkAndUpdate 检查配置变更并通知监听器
func (c *NacosClient) checkAndUpdate(ctx context.Context) {
	newConfig, err := c.fetchFromNacos(ctx)
	if err != nil {
		return
	}

	c.mu.Lock()
	c.config = newConfig
	listeners := make([]ConfigChangeListener, len(c.listeners))
	copy(listeners, c.listeners)
	c.mu.Unlock()

	// 通知所有监听器
	for _, listener := range listeners {
		listener(newConfig)
	}
}

// Close 关闭客户端，停止配置监听
func (c *NacosClient) Close() {
	if c.watchCancel != nil {
		c.watchCancel()
	}
}

// MarshalConfig 序列化配置为 JSON（用于调试/日志）
func (c *AppConfig) MarshalConfig() ([]byte, error) {
	return json.Marshal(c)
}
