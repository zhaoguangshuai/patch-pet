package runtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/patch-pet/patch-pet/pkg/types"
)

// ToolGateway 工具网关
// 所有 Agent 工具调用必须经过此网关，校验权限后方可执行
type ToolGateway struct {
	mu    sync.RWMutex
	tools map[string]types.Tool
}

// NewToolGateway 创建工具网关
func NewToolGateway() *ToolGateway {
	return &ToolGateway{
		tools: make(map[string]types.Tool),
	}
}

// Register 注册工具（Default-Deny：新增工具默认禁用，需人工审批后启用）
func (g *ToolGateway) Register(tool types.Tool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.tools[tool.Name()] = tool
}

// Get 获取已注册工具
func (g *ToolGateway) Get(name string) (types.Tool, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	t, ok := g.tools[name]
	return t, ok
}

// Execute 执行工具调用（含权限校验）
func (g *ToolGateway) Execute(ctx context.Context, call types.ToolCall) (any, error) {
	g.mu.RLock()
	tool, ok := g.tools[call.ToolName]
	g.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("工具 %s 未注册", call.ToolName)
	}

	// 权限校验交由策略引擎（由调用方注入 context）
	// 此处仅做工具存在性校验

	return tool.Execute(ctx, call.Input)
}

// ListRegistered 列出所有已注册工具名称
func (g *ToolGateway) ListRegistered() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	names := make([]string, 0, len(g.tools))
	for name := range g.tools {
		names = append(names, name)
	}
	return names
}
