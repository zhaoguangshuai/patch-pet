// Package runtime Agent Runtime 执行引擎
// 固定链路：LLM → ToolIntent → Tool Gateway → Policy Engine → Execute
// Agent 禁止直连数据库或第三方，必须通过工具网关 + 策略引擎双重校验
package runtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/patch-pet/patch-pet/pkg/constants"
	"github.com/patch-pet/patch-pet/pkg/types"
)

// AgentRuntime Agent 运行时，管控工具调用链路与阈值守护
type AgentRuntime struct {
	mu                sync.Mutex
	toolCallsInSession int // 当前会话工具调用计数
	maxToolCalls      int
	maxNestingDepth   int
}

// New 创建 AgentRuntime 实例
func New() *AgentRuntime {
	return &AgentRuntime{
		maxToolCalls:    constants.MaxToolCallsPerSession,
		maxNestingDepth: constants.MaxToolNestingDepth,
	}
}

// ResetSession 重置会话计数（新会话开始时调用）
func (r *AgentRuntime) ResetSession() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.toolCallsInSession = 0
}

// ValidateToolCall 校验工具调用是否在阈值内
func (r *AgentRuntime) ValidateToolCall() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.toolCallsInSession >= r.maxToolCalls {
		return fmt.Errorf("单会话工具调用数已达上限 %d，强制降级", r.maxToolCalls)
	}
	r.toolCallsInSession++
	return nil
}

// ValidateNestingDepth 校验嵌套深度
func (r *AgentRuntime) ValidateNestingDepth(depth int) error {
	if depth > r.maxNestingDepth {
		return fmt.Errorf("工具嵌套深度 %d 超过上限 %d，强制降级", depth, r.maxNestingDepth)
	}
	return nil
}

// ValidateLLMOutput 校验 LLM 输出 Schema 合规性
// 必须包含：intent, action, risk_level, need_confirm, tool_calls
func ValidateLLMOutput(intent *types.ToolIntent) error {
	if intent == nil {
		return fmt.Errorf("LLM 输出为空")
	}
	if intent.Intent == "" {
		return fmt.Errorf("LLM 输出缺少 intent 字段")
	}
	switch intent.Action {
	case types.LLMActionExecuteTool, types.LLMActionNotice, types.LLMActionDeny:
	default:
		return fmt.Errorf("LLM 输出 action 不合规: %s", intent.Action)
	}
	switch intent.RiskLevel {
	case types.RiskLevelP0, types.RiskLevelP1, types.RiskLevelP2:
	default:
		return fmt.Errorf("LLM 输出 risk_level 不合规: %s", intent.RiskLevel)
	}
	return nil
}

// ToolExecutor 工具执行器，串联 Gateway → Policy → Execute
type ToolExecutor struct {
	gateway *ToolGateway
}

// NewToolExecutor 创建工具执行器
func NewToolExecutor(gateway *ToolGateway) *ToolExecutor {
	return &ToolExecutor{gateway: gateway}
}

// Execute 执行工具调用（含权限校验）
func (e *ToolExecutor) Execute(ctx context.Context, call types.ToolCall) (any, error) {
	return e.gateway.Execute(ctx, call)
}
