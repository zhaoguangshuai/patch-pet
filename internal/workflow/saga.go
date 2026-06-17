// Package workflow Saga 分布式事务编排
// 代遛订单主流程：创建订单 → 支付 → 预约服务商 → 启动服务 → 完成
// 异常补偿：预约失败自动原路退款；回调超时失败关单退款；服务中断回滚至「已预约」
package workflow

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/patch-pet/patch-pet/internal/audit"
	"github.com/patch-pet/patch-pet/internal/lifeflow"
	"github.com/patch-pet/patch-pet/pkg/logger"
)

// SagaStep Saga 步骤状态
type SagaStepStatus string

const (
	StepPending    SagaStepStatus = "pending"    // 待执行
	StepRunning    SagaStepStatus = "running"    // 执行中
	StepCompleted  SagaStepStatus = "completed"  // 已完成
	StepFailed     SagaStepStatus = "failed"     // 执行失败
	StepCompensated SagaStepStatus = "compensated" // 已补偿
)

// SagaStep 定义一个 Saga 步骤
type SagaStep struct {
	Name       string                                          // 步骤名称
	Execute    func(ctx context.Context, state *SagaState) error // 正向执行
	Compensate func(ctx context.Context, state *SagaState) error // 补偿回滚
}

// SagaState Saga 执行状态（在步骤间传递的上下文）
type SagaState struct {
	OrderID         string `json:"order_id"`
	PaymentOrderID  string `json:"payment_order_id"`
	VendorOrderID   string `json:"vendor_order_id"`
	CurrentStep     int    `json:"current_step"`
	StepStatuses    []SagaStepStatus `json:"step_statuses"`
	Error           error  `json:"-"`
}

// Saga Saga 编排器
type Saga struct {
	name     string
	steps    []SagaStep
	lifeflow lifeflow.Writer
	auditLog audit.Recorder
}

// NewSaga 创建 Saga 编排器
func NewSaga(name string, lf lifeflow.Writer, ar audit.Recorder) *Saga {
	return &Saga{
		name:     name,
		lifeflow: lf,
		auditLog: ar,
	}
}

// AddStep 添加 Saga 步骤
func (s *Saga) AddStep(step SagaStep) {
	s.steps = append(s.steps, step)
}

// Execute 执行 Saga（正向执行，失败时自动补偿）
func (s *Saga) Execute(ctx context.Context, state *SagaState) error {
	state.StepStatuses = make([]SagaStepStatus, len(s.steps))
	for i := range state.StepStatuses {
		state.StepStatuses[i] = StepPending
	}

	// 正向执行
	for i, step := range s.steps {
		state.CurrentStep = i
		state.StepStatuses[i] = StepRunning

		if s.lifeflow != nil {
			_ = lifeflow.WriteAgentAction(ctx, s.lifeflow, state.OrderID, "saga", step.Name,
				map[string]any{"step": i, "action": "execute"})
		}

		if err := step.Execute(ctx, state); err != nil {
			state.StepStatuses[i] = StepFailed
			state.Error = err

			logger.Error("Saga 步骤执行失败",
				zap.String("saga", s.name),
				zap.String("step", step.Name),
				zap.Int("step_index", i),
				zap.Error(err),
			)

			// 写审计日志
			if s.auditLog != nil {
				_ = s.auditLog.Record(ctx, audit.Entry{
					Action:   audit.ActionSagaCompensate,
					Level:    audit.LevelWarn,
					Module:   "saga",
					EntityID: state.OrderID,
					Detail: map[string]any{
						"saga":        s.name,
						"failed_step": step.Name,
						"error":       err.Error(),
					},
				})
			}

			// 触发补偿（逆序回滚已执行的步骤）
			s.compensate(ctx, state, i-1)
			return fmt.Errorf("Saga %s 步骤 %s 失败: %w", s.name, step.Name, err)
		}

		state.StepStatuses[i] = StepCompleted
	}

	if s.lifeflow != nil {
		_ = lifeflow.WriteAgentAction(ctx, s.lifeflow, state.OrderID, "saga", "completed",
			map[string]any{"saga": s.name, "steps": len(s.steps)})
	}

	return nil
}

// compensate 逆序补偿已执行的步骤
func (s *Saga) compensate(ctx context.Context, state *SagaState, fromStep int) {
	for i := fromStep; i >= 0; i-- {
		step := s.steps[i]
		if state.StepStatuses[i] != StepCompleted {
			continue
		}

		logger.Info("Saga 补偿执行",
			zap.String("saga", s.name),
			zap.String("step", step.Name),
			zap.Int("step_index", i),
		)

		if step.Compensate != nil {
			if err := step.Compensate(ctx, state); err != nil {
				logger.Error("Saga 补偿失败（需人工介入）",
					zap.String("saga", s.name),
					zap.String("step", step.Name),
					zap.Error(err),
				)

				// 补偿失败 → 审计 + HITL
				if s.auditLog != nil {
					_ = s.auditLog.Record(ctx, audit.Entry{
						Action:   audit.ActionHITLTrigger,
						Level:    audit.LevelError,
						Module:   "saga",
						EntityID: state.OrderID,
						Detail: map[string]any{
							"saga":            s.name,
							"compensate_step": step.Name,
							"error":           err.Error(),
							"reason":          "补偿失败，需人工介入",
						},
					})
				}
			}
		}

		state.StepStatuses[i] = StepCompensated
	}
}

// DogWalkOrderSaga 创建代遛订单 Saga
// 正向：创建订单 → 支付 → 预约服务商 → 启动服务 → 完成
// 补偿：预约失败→退款；服务中断→回滚至已预约
func DogWalkOrderSaga(lf lifeflow.Writer, ar audit.Recorder) *Saga {
	saga := NewSaga("dogwalk_order", lf, ar)

	saga.AddStep(SagaStep{
		Name: "create_order",
		Execute: func(ctx context.Context, state *SagaState) error {
			// 订单创建逻辑（由 handler 层完成，此处为 Saga 编排占位）
			logger.Info("Saga: 创建订单", zap.String("order_id", state.OrderID))
			return nil
		},
		Compensate: func(ctx context.Context, state *SagaState) error {
			// 取消订单
			logger.Info("Saga 补偿: 取消订单", zap.String("order_id", state.OrderID))
			return nil
		},
	})

	saga.AddStep(SagaStep{
		Name: "payment",
		Execute: func(ctx context.Context, state *SagaState) error {
			logger.Info("Saga: 发起支付", zap.String("order_id", state.OrderID))
			return nil
		},
		Compensate: func(ctx context.Context, state *SagaState) error {
			// 原路退款
			logger.Info("Saga 补偿: 退款", zap.String("order_id", state.OrderID),
				zap.String("payment_order_id", state.PaymentOrderID))
			return nil
		},
	})

	saga.AddStep(SagaStep{
		Name: "book_vendor",
		Execute: func(ctx context.Context, state *SagaState) error {
			logger.Info("Saga: 预约服务商", zap.String("order_id", state.OrderID))
			return nil
		},
		Compensate: func(ctx context.Context, state *SagaState) error {
			// 取消预约 + 退款
			logger.Info("Saga 补偿: 取消预约并退款", zap.String("order_id", state.OrderID))
			return nil
		},
	})

	saga.AddStep(SagaStep{
		Name: "start_service",
		Execute: func(ctx context.Context, state *SagaState) error {
			logger.Info("Saga: 启动服务", zap.String("order_id", state.OrderID))
			return nil
		},
		Compensate: func(ctx context.Context, state *SagaState) error {
			// 回滚至已预约状态
			logger.Info("Saga 补偿: 回滚至已预约", zap.String("order_id", state.OrderID))
			return nil
		},
	})

	saga.AddStep(SagaStep{
		Name: "complete",
		Execute: func(ctx context.Context, state *SagaState) error {
			logger.Info("Saga: 服务完成", zap.String("order_id", state.OrderID))
			return nil
		},
		Compensate: nil, // 完成步骤无需补偿
	})

	return saga
}
