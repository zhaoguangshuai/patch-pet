package types

// RiskLevel 风险等级枚举
type RiskLevel string

const (
	RiskLevelP0 RiskLevel = "P0" // 高危：医疗外发、跨家庭授权、大额支付
	RiskLevelP1 RiskLevel = "P1" // 中危：服务商预约、Saga 异常
	RiskLevelP2 RiskLevel = "P2" // 低危：普通工具调用、Token 超阈
)

// LLMAction LLM 输出动作枚举
type LLMAction string

const (
	LLMActionExecuteTool LLMAction = "execute_tool" // 执行工具调用
	LLMActionNotice      LLMAction = "notice"       // 通知用户
	LLMActionDeny        LLMAction = "deny"         // 拒绝执行
)

// ToolIntent LLM 输出结构（强校验）
// 不合规输出由安全网关直接拦截，记录至审计日志
type ToolIntent struct {
	Intent     string     `json:"intent"`      // 意图描述
	Action     LLMAction  `json:"action"`      // 动作：execute_tool / notice / deny
	RiskLevel  RiskLevel  `json:"risk_level"`  // 风险等级：P0 / P1 / P2
	NeedConfirm bool      `json:"need_confirm"` // 是否需要用户确认
	ToolCalls  []ToolCall `json:"tool_calls"`  // 工具调用列表
}

// ToolCall 单次工具调用请求
type ToolCall struct {
	ToolName string         `json:"tool_name"` // 工具名称
	Input    map[string]any `json:"input"`     // 工具入参
}
