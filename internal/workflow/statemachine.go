// Package workflow Saga 状态机引擎
// 无状态状态机，流转配置化，禁止手写大量 switch-case
// 状态机不允许跨阶跳变；已关闭/归档的疗程或订单仅可查询
package workflow

import (
	"fmt"
	"sync"
)

// Transition 状态转换定义
type Transition struct {
	From       string // 当前状态
	To         string // 目标状态
	Event      string // 触发事件
	GuardCheck func(ctx map[string]any) error // 前置校验（权限/幂等键/业务规则）
}

// StateMachine 通用状态机引擎
// 配置化流转，支持前置校验，禁止跨阶跳变
type StateMachine struct {
	mu          sync.RWMutex
	transitions map[string][]Transition // key: fromState
	finalStates map[string]bool         // 终态集合，不可再变更
}

// NewStateMachine 创建状态机实例
func NewStateMachine() *StateMachine {
	return &StateMachine{
		transitions: make(map[string][]Transition),
		finalStates: make(map[string]bool),
	}
}

// AddTransition 注册一条状态转换规则
func (sm *StateMachine) AddTransition(t Transition) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.transitions[t.From] = append(sm.transitions[t.From], t)
}

// MarkFinal 标记终态（已关闭/归档状态不可再变更）
func (sm *StateMachine) MarkFinal(state string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.finalStates[state] = true
}

// CanTransition 检查是否允许从当前状态转换到目标状态
func (sm *StateMachine) CanTransition(from, to string, ctx map[string]any) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 终态不可再变更
	if sm.finalStates[from] {
		return fmt.Errorf("状态 %s 为终态，禁止二次变更", from)
	}

	transitions, ok := sm.transitions[from]
	if !ok {
		return fmt.Errorf("状态 %s 无任何出边", from)
	}

	for _, t := range transitions {
		if t.To == to {
			if t.GuardCheck != nil {
				if err := t.GuardCheck(ctx); err != nil {
					return fmt.Errorf("前置校验失败: %w", err)
				}
			}
			return nil
		}
	}

	return fmt.Errorf("禁止跨阶跳变: %s → %s", from, to)
}

// ExecuteTransition 执行状态转换（校验 + 返回目标状态）
func (sm *StateMachine) ExecuteTransition(from, to string, ctx map[string]any) (string, error) {
	if err := sm.CanTransition(from, to, ctx); err != nil {
		return "", err
	}
	return to, nil
}
