// Package lifeflow 生命流事件服务
// 所有 Agent 操作、异常、执行结果统一写入生命流
package lifeflow

import (
	"context"

	"github.com/patch-pet/patch-pet/pkg/types"
)

// EventType 生命流事件类型
type EventType string

const (
	EventAgentAction    EventType = "agent.action"     // Agent 执行动作
	EventAgentError     EventType = "agent.error"      // Agent 执行异常
	EventStateChange    EventType = "state.change"     // 状态变更
	EventToolCall       EventType = "tool.call"        // 工具调用
	EventSafetyAlert    EventType = "safety.alert"     // 安全告警
	EventUserConfirm    EventType = "user.confirm"     // 用户确认
	EventSystemNotify   EventType = "system.notify"    // 系统通知
)

// Event 生命流事件
type Event struct {
	ID          string         `json:"id"`
	EventType   EventType      `json:"event_type"`
	AggregateID string         `json:"aggregate_id"` // 聚合根 ID（疗程/订单等）
	TraceID     string         `json:"trace_id"`
	Module      string         `json:"module"`
	Actor       string         `json:"actor"` // 操作者（user/agent/system）
	Detail      map[string]any `json:"detail"`
	CreatedAt   types.CSTTime  `json:"created_at"`
}

// Writer 生命流写入器接口
type Writer interface {
	// Write 写入生命流事件
	Write(ctx context.Context, event Event) error
}
