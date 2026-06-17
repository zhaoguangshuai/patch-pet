// Package hitl Human-In-The-Loop 触发与工单
// 触发条件：P0 风险、Agent 连续失败、用户投诉、权限冲突
// 工单状态：Open → InProgress → Resolved / Escalated
package hitl

import (
	"context"
	"fmt"
	"sync"

	"github.com/patch-pet/patch-pet/internal/audit"
	"github.com/patch-pet/patch-pet/pkg/types"
	"github.com/patch-pet/patch-pet/pkg/utils"
)

// TriggerType 触发类型
type TriggerType string

const (
	TriggerP0Risk          TriggerType = "p0_risk"           // P0 高危操作
	TriggerAgentConsecutiveFail TriggerType = "agent_consecutive_fail" // Agent 连续失败
	TriggerUserComplaint    TriggerType = "user_complaint"    // 用户投诉
	TriggerPermissionConflict TriggerType = "permission_conflict" // 权限冲突
	TriggerSagaFailure      TriggerType = "saga_failure"      // Saga 补偿失败
	TriggerEscalation       TriggerType = "escalation"        // 升级触发
)

// WorkOrderStatus 工单状态
type WorkOrderStatus string

const (
	StatusOpen       WorkOrderStatus = "open"
	StatusInProgress  WorkOrderStatus = "in_progress"
	StatusResolved    WorkOrderStatus = "resolved"
	StatusEscalated   WorkOrderStatus = "escalated"
)

// WorkOrder HITL 工单
type WorkOrder struct {
	ID          string          `json:"id"`
	TriggerType TriggerType     `json:"trigger_type"`
	Priority    types.RiskLevel `json:"priority"`
	Module      string          `json:"module"`
	Description string          `json:"description"`
	Context     map[string]any  `json:"context,omitempty"`
	Status      WorkOrderStatus `json:"status"`
	Assignee    string          `json:"assignee,omitempty"`
	Resolution  string          `json:"resolution,omitempty"`
	CreatedAt   types.CSTTime   `json:"created_at"`
	UpdatedAt   types.CSTTime   `json:"updated_at"`
}

// Trigger 触发 HITL 工单
type Trigger struct {
	mu        sync.Mutex
	orders    map[string]*WorkOrder
	auditLog  audit.Recorder
	failCount map[string]int // module → consecutive fail count
}

// NewTrigger 创建 HITL 触发器
func NewTrigger(auditLog audit.Recorder) *Trigger {
	return &Trigger{
		orders:    make(map[string]*WorkOrder),
		auditLog:  auditLog,
		failCount: make(map[string]int),
	}
}

// Fire 触发 HITL 工单
func (t *Trigger) Fire(ctx context.Context, triggerType TriggerType, module string, description string, contextData map[string]any) (*WorkOrder, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	order := &WorkOrder{
		ID:          utils.GenerateULID(types.IDPrefix("hitl")),
		TriggerType: triggerType,
		Priority:    t.resolvePriority(triggerType),
		Module:      module,
		Description: description,
		Context:     contextData,
		Status:      StatusOpen,
		CreatedAt:   types.NowCST(),
		UpdatedAt:   types.NowCST(),
	}

	t.orders[order.ID] = order

	// 审计记录
	if t.auditLog != nil {
		t.auditLog.Record(ctx, audit.Entry{
			Action:  audit.ActionHITLTrigger,
			Level:   audit.LevelWarn,
			Module:  module,
			EntityID: order.ID,
			Detail: map[string]any{
				"trigger_type": string(triggerType),
				"description":  description,
			},
		})
	}

	return order, nil
}

// RecordAgentFailure 记录 Agent 失败，连续 3 次自动触发 HITL
func (t *Trigger) RecordAgentFailure(ctx context.Context, module string, err error) (*WorkOrder, error) {
	t.mu.Lock()
	t.failCount[module]++
	count := t.failCount[module]
	t.mu.Unlock()

	if count >= 3 {
		return t.Fire(ctx, TriggerAgentConsecutiveFail, module,
			fmt.Sprintf("Agent [%s] 连续失败 %d 次: %v", module, count, err),
			map[string]any{"fail_count": count, "error": err.Error()},
		)
	}
	return nil, nil
}

// ResetAgentFailures 重置 Agent 失败计数（成功时调用）
func (t *Trigger) ResetAgentFailures(module string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.failCount[module] = 0
}

// GetOrder 获取工单
func (t *Trigger) GetOrder(id string) (*WorkOrder, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	order, ok := t.orders[id]
	return order, ok
}

// ListOpenOrders 列出所有未关闭工单
func (t *Trigger) ListOpenOrders() []*WorkOrder {
	t.mu.Lock()
	defer t.mu.Unlock()

	var result []*WorkOrder
	for _, o := range t.orders {
		if o.Status == StatusOpen || o.Status == StatusInProgress {
			result = append(result, o)
		}
	}
	return result
}

// UpdateStatus 更新工单状态
func (t *Trigger) UpdateStatus(id string, status WorkOrderStatus, assignee string, resolution string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	order, ok := t.orders[id]
	if !ok {
		return fmt.Errorf("工单不存在: %s", id)
	}

	order.Status = status
	if assignee != "" {
		order.Assignee = assignee
	}
	if resolution != "" {
		order.Resolution = resolution
	}
	order.UpdatedAt = types.NowCST()
	return nil
}

func (t *Trigger) resolvePriority(triggerType TriggerType) types.RiskLevel {
	switch triggerType {
	case TriggerP0Risk, TriggerSagaFailure:
		return types.RiskLevelP0
	case TriggerAgentConsecutiveFail, TriggerPermissionConflict:
		return types.RiskLevelP1
	default:
		return types.RiskLevelP2
	}
}
