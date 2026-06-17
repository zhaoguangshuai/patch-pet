// Package agent Agent 编排层
// 管理医疗 Agent（16 号）与代遛 Agent（19 号）的生命周期与调度
package agent

import (
	"context"

	"github.com/patch-pet/patch-pet/pkg/types"
)

// AgentType Agent 类型
type AgentType string

const (
	AgentTypeMedical AgentType = "medical" // 16 号 医疗居家 Agent
	AgentTypeDogwalk AgentType = "dogwalk" // 19 号 代遛狗 Agent
)

// Agent Agent 实例接口
type Agent interface {
	// Type Agent 类型
	Type() AgentType
	// Handle 处理用户请求，返回 ToolIntent
	Handle(ctx context.Context, input string) (*types.ToolIntent, error)
}
