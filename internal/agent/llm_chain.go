// Package agent LLM 降级链路
// GPT-5 → Claude Sonnet → Gemini → 静态模板
// 所有降级强制日志记录
package agent

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/patch-pet/patch-pet/internal/audit"
	"github.com/patch-pet/patch-pet/pkg/logger"
)

// LLMProvider LLM 供应商
type LLMProvider string

const (
	ProviderGPT5         LLMProvider = "gpt-5"          // OpenAI GPT-5
	ProviderClaudeSonnet LLMProvider = "claude-sonnet"  // Anthropic Claude Sonnet
	ProviderGemini       LLMProvider = "gemini"         // Google Gemini
	ProviderStatic       LLMProvider = "static"         // 静态模板（最终兜底）
)

// LLMRequest LLM 请求
type LLMRequest struct {
	Prompt      string         `json:"prompt"`
	MaxTokens   int            `json:"max_tokens"`
	Temperature float64        `json:"temperature"`
	Extra       map[string]any `json:"extra,omitempty"`
}

// LLMResponse LLM 响应
type LLMResponse struct {
	Content    string      `json:"content"`
	Provider   LLMProvider `json:"provider"`
	TokenUsed  int         `json:"token_used"`
	IsFallback bool        `json:"is_fallback"` // 是否为降级响应
}

// LLMCaller LLM 调用器接口
type LLMCaller interface {
	// Call 调用 LLM，失败返回 error
	Call(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
	// Provider 返回供应商名称
	Provider() LLMProvider
}

// DegradationChain LLM 降级链路
// 按优先级依次尝试：GPT-5 → Claude Sonnet → Gemini → 静态模板
type DegradationChain struct {
	callers   []LLMCaller
	auditLog  audit.Recorder
	staticTpl string // 静态模板内容
}

// NewDegradationChain 创建降级链路
func NewDegradationChain(ar audit.Recorder, staticTemplate string) *DegradationChain {
	return &DegradationChain{
		auditLog:  ar,
		staticTpl: staticTemplate,
	}
}

// AddCaller 添加 LLM 调用器（按优先级顺序添加）
func (c *DegradationChain) AddCaller(caller LLMCaller) {
	c.callers = append(c.callers, caller)
}

// Call 执行 LLM 调用（带自动降级）
// 依次尝试每个供应商，全部失败后使用静态模板
func (c *DegradationChain) Call(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	var lastErr error

	for _, caller := range c.callers {
		resp, err := caller.Call(ctx, req)
		if err != nil {
			lastErr = err

			logger.Warn("LLM 调用失败，准备降级",
				zap.String("provider", string(caller.Provider())),
				zap.Error(err),
			)

			// 写审计日志（降级事件）
			if c.auditLog != nil {
				_ = c.auditLog.Record(ctx, audit.Entry{
					Action: audit.ActionLLMDegraded,
					Level:  audit.LevelWarn,
					Module: "agent",
					Detail: map[string]any{
						"failed_provider": string(caller.Provider()),
						"error":           err.Error(),
					},
				})
			}

			continue
		}

		// 成功
		return resp, nil
	}

	// 所有 LLM 供应商均失败，使用静态模板
	logger.Error("所有 LLM 供应商均失败，使用静态模板",
		zap.Int("tried_count", len(c.callers)),
		zap.Error(lastErr),
	)

	if c.auditLog != nil {
		_ = c.auditLog.Record(ctx, audit.Entry{
			Action: audit.ActionLLMDegraded,
			Level:  audit.LevelError,
			Module: "agent",
			Detail: map[string]any{
				"final_fallback": "static_template",
				"error":          fmt.Sprintf("所有 %d 个 LLM 供应商均失败: %v", len(c.callers), lastErr),
			},
		})
	}

	return &LLMResponse{
		Content:    c.staticTpl,
		Provider:   ProviderStatic,
		TokenUsed:  0,
		IsFallback: true,
	}, nil
}

// MockCaller 模拟 LLM 调用器（用于测试）
type MockCaller struct {
	provider   LLMProvider
	response   *LLMResponse
	err        error
	callCount  int
}

// NewMockCaller 创建模拟调用器
func NewMockCaller(provider LLMProvider, response *LLMResponse, err error) *MockCaller {
	return &MockCaller{
		provider: provider,
		response: response,
		err:      err,
	}
}

// Call 模拟调用
func (m *MockCaller) Call(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

// Provider 返回供应商名称
func (m *MockCaller) Provider() LLMProvider {
	return m.provider
}

// CallCount 返回调用次数
func (m *MockCaller) CallCount() int {
	return m.callCount
}
