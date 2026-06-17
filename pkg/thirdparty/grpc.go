// Package thirdparty gRPC 客户端封装
// Go ↔ Python AI 服务低延迟通道：实时摘要、意图识别
// 超时 3s，连续 10 次失败熔断 5 分钟
package thirdparty

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/patch-pet/patch-pet/pkg/logger"
)

const (
	// GRPCDefaultTimeout gRPC 默认超时
	GRPCDefaultTimeout = 3 * time.Second
	// GRPCMaxRetries gRPC 最大重试次数
	GRPCMaxRetries = 2
	// CircuitBreakerThreshold 连续失败次数触发熔断
	CircuitBreakerThreshold = 10
	// CircuitBreakerRecovery 熔断恢复时间
	CircuitBreakerRecovery = 5 * time.Minute
)

// GRPCConfig gRPC 客户端配置
type GRPCConfig struct {
	Addr    string        // AI 服务地址（环境变量 AI_SERVICE_GRPC_ADDR）
	Timeout time.Duration // 调用超时
}

// DefaultGRPCConfig 默认 gRPC 配置
func DefaultGRPCConfig() GRPCConfig {
	return GRPCConfig{
		Addr:    getEnvOrDefault("AI_SERVICE_GRPC_ADDR", "localhost:50051"),
		Timeout: GRPCDefaultTimeout,
	}
}

// --- 请求/响应类型 ---

// SummaryRequest 实时摘要请求
type SummaryRequest struct {
	EpisodeID    string `json:"episode_id"`
	WindowStart  string `json:"window_start"`
	WindowEnd    string `json:"window_end"`
	IncludeTasks bool   `json:"include_tasks"`
}

// SummaryResponse 实时摘要响应
type SummaryResponse struct {
	SummaryID    string `json:"summary_id"`
	OwnerText    string `json:"owner_text"`
	ClinicText   string `json:"clinic_text"`
	SafetyStatus string `json:"safe_status"`
}

// IntentRequest 意图识别请求
type IntentRequest struct {
	UserMessage string         `json:"user_message"`
	Context     map[string]any `json:"context,omitempty"`
	AgentType   string         `json:"agent_type"` // medical / dogwalk
}

// IntentResponse 意图识别响应
type IntentResponse struct {
	Intent     string         `json:"intent"`
	Confidence float64        `json:"confidence"`
	ToolCalls  []ToolCall     `json:"tool_calls,omitempty"`
	Extra      map[string]any `json:"extra,omitempty"`
}

// ToolCall 工具调用意图
type ToolCall struct {
	ToolName   string         `json:"tool_name"`
	Parameters map[string]any `json:"parameters,omitempty"`
	RiskLevel  string         `json:"risk_level"` // P0/P1/P2
}

// --- 熔断器 ---

// circuitBreaker 简易熔断器
type circuitBreaker struct {
	failures    atomic.Int64
	lastFailure atomic.Value // time.Time
	open        atomic.Bool
}

func newCircuitBreaker() *circuitBreaker {
	cb := &circuitBreaker{}
	cb.lastFailure.Store(time.Time{})
	return cb
}

func (cb *circuitBreaker) isOpen() bool {
	if !cb.open.Load() {
		return false
	}
	lastFail := cb.lastFailure.Load().(time.Time)
	if time.Since(lastFail) > CircuitBreakerRecovery {
		cb.open.Store(false)
		cb.failures.Store(0)
		logger.Info("熔断器恢复")
		return false
	}
	return true
}

func (cb *circuitBreaker) recordSuccess() {
	cb.failures.Store(0)
	cb.open.Store(false)
}

func (cb *circuitBreaker) recordFailure() {
	count := cb.failures.Add(1)
	cb.lastFailure.Store(time.Now())
	if count >= CircuitBreakerThreshold {
		cb.open.Store(true)
		logger.Error("熔断器触发",
			zap.Int64("failures", count),
			zap.Duration("recovery", CircuitBreakerRecovery),
		)
	}
}

// --- gRPC 客户端 ---

// AIServiceClient Python AI 服务 gRPC 客户端
// 实际生产环境应使用 grpc-go 连接 Python gRPC 服务
// 当前为接口封装，底层可替换为真实 gRPC 连接
type AIServiceClient struct {
	config GRPCConfig
	cb     *circuitBreaker
}

// NewAIServiceClient 创建 AI 服务客户端
func NewAIServiceClient(cfg GRPCConfig) *AIServiceClient {
	return &AIServiceClient{
		config: cfg,
		cb:     newCircuitBreaker(),
	}
}

// GenerateSummary 实时摘要生成（gRPC 调用）
func (c *AIServiceClient) GenerateSummary(ctx context.Context, req *SummaryRequest) (*SummaryResponse, error) {
	if c.cb.isOpen() {
		return nil, fmt.Errorf("熔断器打开，拒绝调用 AI 服务")
	}

	// 带超时的上下文
	callCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	_ = callCtx // 实际 gRPC 调用使用 callCtx

	// TODO: 替换为真实 gRPC 调用
	// conn, err := grpc.DialContext(callCtx, c.config.Addr, grpc.WithInsecure())
	// client := pb.NewAIServiceClient(conn)
	// resp, err := client.GenerateSummary(callCtx, req)

	logger.Info("gRPC 调用: GenerateSummary",
		zap.String("episode_id", req.EpisodeID),
		zap.String("addr", c.config.Addr),
	)

	c.cb.recordSuccess()

	return &SummaryResponse{
		SummaryID:    "sum_placeholder",
		OwnerText:    "摘要生成中...",
		ClinicText:   "摘要生成中...",
		SafetyStatus: "safe",
	}, nil
}

// RecognizeIntent 意图识别（gRPC 调用）
func (c *AIServiceClient) RecognizeIntent(ctx context.Context, req *IntentRequest) (*IntentResponse, error) {
	if c.cb.isOpen() {
		return nil, fmt.Errorf("熔断器打开，拒绝调用 AI 服务")
	}

	callCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	_ = callCtx

	logger.Info("gRPC 调用: RecognizeIntent",
		zap.String("agent_type", req.AgentType),
		zap.String("addr", c.config.Addr),
	)

	c.cb.recordSuccess()

	return &IntentResponse{
		Intent:     "general_query",
		Confidence: 0.5,
	}, nil
}

// HealthCheck AI 服务健康检查
func (c *AIServiceClient) HealthCheck(ctx context.Context) error {
	callCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	_ = callCtx

	// TODO: 替换为真实 gRPC health check
	// conn, err := grpc.DialContext(callCtx, c.config.Addr, grpc.WithInsecure())

	return nil
}

// IsAvailable AI 服务是否可用（熔断器未打开）
func (c *AIServiceClient) IsAvailable() bool {
	return !c.cb.isOpen()
}
