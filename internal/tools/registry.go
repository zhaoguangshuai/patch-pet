// Package tools 工具注册中心
// 统一管理所有 Agent 可调用工具，Default-Deny 策略
package tools

import (
	"sync"

	"github.com/patch-pet/patch-pet/pkg/types"
)

// Registry 工具注册中心
type Registry struct {
	mu    sync.RWMutex
	tools map[string]types.Tool
}

// New 创建工具注册中心
func New() *Registry {
	return &Registry{
		tools: make(map[string]types.Tool),
	}
}

// Register 注册工具
func (r *Registry) Register(tool types.Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
}

// Get 获取工具
func (r *Registry) Get(name string) (types.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// List 列出所有已注册工具
func (r *Registry) List() []types.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]types.Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}
