package thirdparty

import (
	"testing"
	"time"
)

func TestGRPCConfigDefaults(t *testing.T) {
	cfg := DefaultGRPCConfig()
	if cfg.Timeout != GRPCDefaultTimeout {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, GRPCDefaultTimeout)
	}
	if cfg.Addr == "" {
		t.Error("Addr should not be empty")
	}
}

func TestGRPCConstants(t *testing.T) {
	if GRPCDefaultTimeout != 3*time.Second {
		t.Errorf("GRPCDefaultTimeout = %v, want 3s", GRPCDefaultTimeout)
	}
	if GRPCMaxRetries != 2 {
		t.Errorf("GRPCMaxRetries = %d, want 2", GRPCMaxRetries)
	}
	if CircuitBreakerThreshold != 10 {
		t.Errorf("CircuitBreakerThreshold = %d, want 10", CircuitBreakerThreshold)
	}
	if CircuitBreakerRecovery != 5*time.Minute {
		t.Errorf("CircuitBreakerRecovery = %v, want 5m", CircuitBreakerRecovery)
	}
}

func TestAIServiceClientCreation(t *testing.T) {
	cfg := DefaultGRPCConfig()
	client := NewAIServiceClient(cfg)
	if client == nil {
		t.Fatal("NewAIServiceClient should not return nil")
	}
	if !client.IsAvailable() {
		t.Error("Client should be available initially")
	}
}

func TestCircuitBreakerInitiallyClosed(t *testing.T) {
	cb := newCircuitBreaker()
	if cb.isOpen() {
		t.Error("Circuit breaker should be closed initially")
	}
}

func TestCircuitBreakerOpensAfterThreshold(t *testing.T) {
	cb := newCircuitBreaker()
	for i := 0; i < CircuitBreakerThreshold; i++ {
		cb.recordFailure()
	}
	if !cb.isOpen() {
		t.Error("Circuit breaker should be open after threshold failures")
	}
}

func TestCircuitBreakerResetsOnSuccess(t *testing.T) {
	cb := newCircuitBreaker()
	for i := 0; i < 5; i++ {
		cb.recordFailure()
	}
	cb.recordSuccess()
	if cb.isOpen() {
		t.Error("Circuit breaker should be closed after success")
	}
}

func TestSummaryRequestStruct(t *testing.T) {
	req := &SummaryRequest{
		EpisodeID:    "ep_123",
		WindowStart:  "2026-01-01T00:00:00+08:00",
		WindowEnd:    "2026-01-02T00:00:00+08:00",
		IncludeTasks: true,
	}
	if req.EpisodeID != "ep_123" {
		t.Errorf("EpisodeID = %s, want ep_123", req.EpisodeID)
	}
}

func TestIntentRequestStruct(t *testing.T) {
	req := &IntentRequest{
		UserMessage: "我的狗今天需要遛吗",
		AgentType:   "dogwalk",
	}
	if req.AgentType != "dogwalk" {
		t.Errorf("AgentType = %s, want dogwalk", req.AgentType)
	}
}

func TestToolCallStruct(t *testing.T) {
	tc := &ToolCall{
		ToolName:  "check_task_status",
		RiskLevel: "P1",
	}
	if tc.RiskLevel != "P1" {
		t.Errorf("RiskLevel = %s, want P1", tc.RiskLevel)
	}
}
