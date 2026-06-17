// Package policy 策略引擎（Casbin/OPA + DSL）
// Default-Deny：新增工具/权限默认禁用，需人工审批
package policy

import (
	"fmt"
	"sync"
)

// Decision 策略决策结果
type Decision struct {
	Allow bool   // 是否放行
	Msg   string // 拒绝原因（放行时为空）
}

// Rule 策略规则
type Rule struct {
	ID       string            // 规则 ID
	Name     string            // 规则名称
	When     map[string]string // 条件匹配
	Action   string            // 动作：allow / deny
	Msg      string            // 拒绝消息
	Priority int               // 优先级（数值越大越优先）
}

// Engine 策略引擎
// 基于规则匹配的权限校验，Default-Deny 策略
type Engine struct {
	mu    sync.RWMutex
	rules []Rule
}

// New 创建策略引擎实例
func New() *Engine {
	return &Engine{}
}

// AddRule 添加策略规则
func (e *Engine) AddRule(rule Rule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = append(e.rules, rule)
	e.sortRules()
}

// sortRules 按优先级降序排列
func (e *Engine) sortRules() {
	for i := 1; i < len(e.rules); i++ {
		for j := i; j > 0 && e.rules[j].Priority > e.rules[j-1].Priority; j-- {
			e.rules[j], e.rules[j-1] = e.rules[j-1], e.rules[j]
		}
	}
}

// Evaluate 评估策略，返回决策结果
// Default-Deny：无匹配规则时默认拒绝
func (e *Engine) Evaluate(ctx map[string]string) Decision {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, rule := range e.rules {
		if e.matchRule(rule, ctx) {
			return Decision{
				Allow: rule.Action == "allow",
				Msg:   rule.Msg,
			}
		}
	}

	// Default-Deny：无匹配规则默认拒绝
	return Decision{
		Allow: false,
		Msg:   fmt.Sprintf("Default-Deny: 无匹配规则，权限 %s 被拒绝", ctx["permission"]),
	}
}

// matchRule 检查规则条件是否全部匹配
func (e *Engine) matchRule(rule Rule, ctx map[string]string) bool {
	for key, expected := range rule.When {
		actual, ok := ctx[key]
		if !ok || actual != expected {
			return false
		}
	}
	return true
}
