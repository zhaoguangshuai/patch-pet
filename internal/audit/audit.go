// Package audit 全链路审计日志
// 医疗外发、下单、支付、第三方对接、授权变更全部留痕
package audit

import (
	"context"

	"github.com/patch-pet/patch-pet/pkg/types"
)

// Action 审计动作类型
type Action string

const (
	ActionMedicalShare     Action = "medical.share"      // 医疗数据外发
	ActionMedicalAuth      Action = "medical.auth"       // 医疗授权变更
	ActionOrderCreate      Action = "order.create"       // 订单创建
	ActionOrderPay         Action = "order.pay"          // 支付
	ActionOrderRefund      Action = "order.refund"       // 退款
	ActionThirdPartyCall   Action = "thirdparty.call"    // 第三方接口调用
	ActionToolExecute      Action = "tool.execute"       // 工具执行
	ActionPolicyDeny       Action = "policy.deny"        // 策略拒绝
	ActionLLMDegraded      Action = "llm.degraded"       // LLM 降级
	ActionSagaCompensate   Action = "saga.compensate"    // Saga 补偿
	ActionHITLTrigger      Action = "hitl.trigger"       // 人工接管触发
	ActionIntentConflict   Action = "intent.conflict"    // LAKI 意图冲突
)

// Level 审计级别
type Level string

const (
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

// Entry 审计日志条目
type Entry struct {
	ID        string         `json:"id"`
	TraceID   string         `json:"trace_id"`
	Action    Action         `json:"action"`
	Level     Level          `json:"level"`
	Module    string         `json:"module"`
	UserID    string         `json:"user_id"`
	EntityID  string         `json:"entity_id"`
	Detail    map[string]any `json:"detail"`
	CreatedAt types.CSTTime  `json:"created_at"`
}

// Recorder 审计记录器接口
type Recorder interface {
	// Record 写入审计日志
	Record(ctx context.Context, entry Entry) error
}
