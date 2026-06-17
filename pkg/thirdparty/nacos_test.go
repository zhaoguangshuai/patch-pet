package thirdparty

import (
	"context"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// 验证全局运行阈值默认值
	if cfg.MaxToolCallsPerSession != 5 {
		t.Errorf("MaxToolCallsPerSession = %d, want 5", cfg.MaxToolCallsPerSession)
	}
	if cfg.MaxToolNestingDepth != 3 {
		t.Errorf("MaxToolNestingDepth = %d, want 3", cfg.MaxToolNestingDepth)
	}
	if cfg.MaxLLMInputTokens != 8000 {
		t.Errorf("MaxLLMInputTokens = %d, want 8000", cfg.MaxLLMInputTokens)
	}
	if cfg.MaxLLMOutputTokens != 1500 {
		t.Errorf("MaxLLMOutputTokens = %d, want 1500", cfg.MaxLLMOutputTokens)
	}
	if cfg.ThirdPartyTimeoutMs != 3000 {
		t.Errorf("ThirdPartyTimeoutMs = %d, want 3000", cfg.ThirdPartyTimeoutMs)
	}
	if cfg.CircuitBreakerThreshold != 10 {
		t.Errorf("CircuitBreakerThreshold = %d, want 10", cfg.CircuitBreakerThreshold)
	}
	if cfg.CircuitBreakerDurationSec != 300 {
		t.Errorf("CircuitBreakerDurationSec = %d, want 300", cfg.CircuitBreakerDurationSec)
	}

	// 验证分页默认值
	if cfg.DefaultPageSize != 20 {
		t.Errorf("DefaultPageSize = %d, want 20", cfg.DefaultPageSize)
	}
	if cfg.MaxPageSize != 50 {
		t.Errorf("MaxPageSize = %d, want 50", cfg.MaxPageSize)
	}

	// 验证 Token 预算默认值
	if cfg.DailyTokenBudget != 100000 {
		t.Errorf("DailyTokenBudget = %d, want 100000", cfg.DailyTokenBudget)
	}
	if cfg.MonthlyTokenBudget != 2000000 {
		t.Errorf("MonthlyTokenBudget = %d, want 2000000", cfg.MonthlyTokenBudget)
	}
	if cfg.MaxCostPerUser != 5000 {
		t.Errorf("MaxCostPerUser = %d, want 5000", cfg.MaxCostPerUser)
	}
}

func TestNewNacosClient(t *testing.T) {
	cfg := NacosConfig{
		Addr:      "localhost",
		Port:      8848,
		Namespace: "dev",
		Group:     "DEFAULT_GROUP",
		DataID:    "patch-pet",
	}

	client := NewNacosClient(cfg)
	if client == nil {
		t.Fatal("NewNacosClient returned nil")
	}

	// 初始配置应为默认值
	config := client.GetConfig()
	if config.MaxToolCallsPerSession != 5 {
		t.Errorf("初始配置 MaxToolCallsPerSession = %d, want 5", config.MaxToolCallsPerSession)
	}
}

func TestNacosClientInitFallback(t *testing.T) {
	cfg := NacosConfig{
		Addr:      "invalid-host",
		Port:      8848,
		Namespace: "dev",
		Group:     "DEFAULT_GROUP",
		DataID:    "patch-pet",
	}

	client := NewNacosClient(cfg)
	ctx := context.Background()

	// Nacos 不可用时应降级为默认配置，不返回错误
	err := client.Init(ctx)
	if err != nil {
		t.Errorf("Init should not return error on Nacos failure, got: %v", err)
	}

	// 配置应为默认值
	config := client.GetConfig()
	if config.MaxToolCallsPerSession != 5 {
		t.Errorf("降级后 MaxToolCallsPerSession = %d, want 5", config.MaxToolCallsPerSession)
	}
}

func TestNacosClientOnConfigChange(t *testing.T) {
	cfg := NacosConfig{
		Addr:      "localhost",
		Port:      8848,
		Namespace: "dev",
		Group:     "DEFAULT_GROUP",
		DataID:    "patch-pet",
	}

	client := NewNacosClient(cfg)

	// 注册监听器
	client.OnConfigChange(func(newConfig *AppConfig) {
		// 配置变更回调
	})

	if len(client.listeners) != 1 {
		t.Errorf("listeners count = %d, want 1", len(client.listeners))
	}
}

func TestMarshalConfig(t *testing.T) {
	cfg := DefaultConfig()
	data, err := cfg.MarshalConfig()
	if err != nil {
		t.Fatalf("MarshalConfig failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("MarshalConfig returned empty data")
	}
}

// ---- 边界场景 ----

func TestDefaultConfigValuesNotZero(t *testing.T) {
	cfg := DefaultConfig()

	// 确保关键配置不为零值
	if cfg.ScheduledTaskTimeoutSec == 0 {
		t.Error("ScheduledTaskTimeoutSec should not be zero")
	}
	if cfg.GracefulShutdownSec == 0 {
		t.Error("GracefulShutdownSec should not be zero")
	}
	if cfg.ReplayToleranceSec == 0 {
		t.Error("ReplayToleranceSec should not be zero")
	}
	if cfg.KafkaMaxRetries == 0 {
		t.Error("KafkaMaxRetries should not be zero")
	}
	if cfg.SessionTTLMiuntes == 0 {
		t.Error("SessionTTLMiuntes should not be zero")
	}
}
