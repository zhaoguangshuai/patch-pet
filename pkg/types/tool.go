// Package types 定义全局公共类型，供所有 internal 模块引用
package types

import "context"

// Permission 工具/能力权限标识
type Permission string

// Tool 统一工具接口，所有 Agent 可调用工具必须实现
// 执行链路：LLM → ToolIntent → Tool Gateway → Policy Engine → Execute
type Tool interface {
	// Name 工具唯一名称，用于注册与路由
	Name() string
	// Description 工具功能描述，供 LLM 意图识别
	Description() string
	// Permission 所需权限标识，由策略引擎校验
	Permission() Permission
	// Execute 执行工具逻辑，input/output 由各工具自行定义
	Execute(ctx context.Context, input any) (any, error)
}
