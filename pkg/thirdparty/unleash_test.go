package thirdparty

import (
	"os"
	"testing"
)

func TestNewFeatureFlagClient(t *testing.T) {
	cfg := UnleashConfig{
		URL:        "http://localhost:4242",
		AppName:    "patch-pet",
		InstanceID: "test-001",
	}

	client := NewFeatureFlagClient(cfg)
	if client == nil {
		t.Fatal("NewFeatureFlagClient returned nil")
	}
	if client.config.URL != cfg.URL {
		t.Errorf("URL = %s, want %s", client.config.URL, cfg.URL)
	}
	if client.config.AppName != cfg.AppName {
		t.Errorf("AppName = %s, want %s", client.config.AppName, cfg.AppName)
	}
	if client.config.InstanceID != cfg.InstanceID {
		t.Errorf("InstanceID = %s, want %s", client.config.InstanceID, cfg.InstanceID)
	}
}

func TestFeatureFlagClientInitFallback(t *testing.T) {
	cfg := UnleashConfig{
		URL:        "http://invalid-host:9999",
		AppName:    "patch-pet-test",
		InstanceID: "test-fallback",
	}

	client := NewFeatureFlagClient(cfg)
	err := client.Init()
	if err != nil {
		t.Errorf("Init should not return error on Unleash failure, got: %v", err)
	}

	// 未就绪时所有 flag 默认关闭
	if client.IsEnabled("any-flag") {
		t.Error("IsEnabled should return false when Unleash is not ready")
	}
}

func TestFeatureFlagClientIsEnabledDefaultFalse(t *testing.T) {
	cfg := UnleashConfig{
		URL:        "http://invalid-host:9999",
		AppName:    "patch-pet-test",
		InstanceID: "test-default-false",
	}

	client := NewFeatureFlagClient(cfg)
	_ = client.Init()

	// Default-Deny：未就绪时所有 flag 返回 false
	flags := []string{
		"medical-agent-enabled",
		"dogwalk-agent-enabled",
		"payment-gateway-enabled",
		"new-tool-capability",
		"emergency-circuit-breaker",
	}
	for _, flag := range flags {
		if client.IsEnabled(flag) {
			t.Errorf("IsEnabled(%q) should return false (Default-Deny)", flag)
		}
	}
}

func TestFeatureFlagClientIsEnabledWithDefault(t *testing.T) {
	cfg := UnleashConfig{
		URL:        "http://invalid-host:9999",
		AppName:    "patch-pet-test",
		InstanceID: "test-with-default",
	}

	client := NewFeatureFlagClient(cfg)
	_ = client.Init()

	// 未就绪时使用显式默认值
	if !client.IsEnabledWithDefault("some-flag", true) {
		t.Error("IsEnabledWithDefault should return default true when not ready")
	}
	if client.IsEnabledWithDefault("some-flag", false) {
		t.Error("IsEnabledWithDefault should return default false when not ready")
	}
}

func TestFeatureFlagClientGetVariantDefault(t *testing.T) {
	cfg := UnleashConfig{
		URL:        "http://invalid-host:9999",
		AppName:    "patch-pet-test",
		InstanceID: "test-variant",
	}

	client := NewFeatureFlagClient(cfg)
	_ = client.Init()

	// 未就绪时返回 disabled 变体
	variant := client.GetVariant("some-flag")
	if variant == nil {
		t.Fatal("GetVariant should not return nil")
	}
	if variant.Name != "disabled" {
		t.Errorf("variant.Name = %q, want %q", variant.Name, "disabled")
	}
	if variant.Enabled {
		t.Error("variant.Enabled should be false when not ready")
	}
}

func TestFeatureFlagClientReady(t *testing.T) {
	cfg := UnleashConfig{
		URL:        "http://invalid-host:9999",
		AppName:    "patch-pet-test",
		InstanceID: "test-ready",
	}

	client := NewFeatureFlagClient(cfg)

	// 初始化前未就绪
	if client.Ready() {
		t.Error("Ready should be false before Init")
	}

	_ = client.Init()

	// 连接失败时仍未就绪
	if client.Ready() {
		t.Error("Ready should be false when Unleash connection fails")
	}
}

// ---- 边界场景 ----

func TestFeatureFlagClientDefaultEnvVars(t *testing.T) {
	// 设置环境变量
	os.Setenv("UNLEASH_URL", "http://env-unleash:4242")
	os.Setenv("UNLEASH_APP_NAME", "env-patch-pet")
	os.Setenv("UNLEASH_INSTANCE_ID", "env-instance-001")
	defer os.Unsetenv("UNLEASH_URL")
	defer os.Unsetenv("UNLEASH_APP_NAME")
	defer os.Unsetenv("UNLEASH_INSTANCE_ID")

	// 空配置应从环境变量读取
	cfg := UnleashConfig{}
	client := NewFeatureFlagClient(cfg)
	_ = client.Init()

	if client.config.URL != "http://env-unleash:4242" {
		t.Errorf("URL from env = %s, want http://env-unleash:4242", client.config.URL)
	}
	if client.config.AppName != "env-patch-pet" {
		t.Errorf("AppName from env = %s, want env-patch-pet", client.config.AppName)
	}
	if client.config.InstanceID != "env-instance-001" {
		t.Errorf("InstanceID from env = %s, want env-instance-001", client.config.InstanceID)
	}
}

func TestFeatureFlagClientMultipleFlags(t *testing.T) {
	cfg := UnleashConfig{
		URL:        "http://invalid-host:9999",
		AppName:    "patch-pet-test",
		InstanceID: "test-multi-flags",
	}

	client := NewFeatureFlagClient(cfg)
	_ = client.Init()

	// 多个 flag 并发查询不应 panic
	flags := []string{
		"flag-1", "flag-2", "flag-3", "flag-4", "flag-5",
	}
	for _, flag := range flags {
		if client.IsEnabled(flag) {
			t.Errorf("IsEnabled(%q) should return false (Default-Deny)", flag)
		}
	}
}
