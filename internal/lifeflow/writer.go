package lifeflow

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"

	"github.com/patch-pet/patch-pet/internal/middleware"
	"github.com/patch-pet/patch-pet/pkg/logger"
	"github.com/patch-pet/patch-pet/pkg/types"
	"github.com/patch-pet/patch-pet/pkg/utils"
)

// LogWriter 基于结构化日志的生命流写入器
type LogWriter struct {
	logger *zap.Logger
}

// NewLogWriter 创建日志生命流写入器
func NewLogWriter() *LogWriter {
	return &LogWriter{
		logger: logger.GetLogger().With(zap.String("component", "lifeflow")),
	}
}

// Write 写入生命流事件
func (w *LogWriter) Write(ctx context.Context, event Event) error {
	// 自动生成 ID
	if event.ID == "" {
		event.ID = utils.GenerateULID(types.IDPrefix("evt"))
	}

	// 自动注入 trace_id
	if event.TraceID == "" {
		event.TraceID = middleware.GetTraceID(ctx)
	}

	// 自动注入时间戳
	if event.CreatedAt.Time.IsZero() {
		event.CreatedAt = types.NowCST()
	}

	detailJSON, _ := json.Marshal(event.Detail)

	fields := []zap.Field{
		zap.String("event_id", event.ID),
		zap.String("event_type", string(event.EventType)),
		zap.String("aggregate_id", event.AggregateID),
		zap.String("trace_id", event.TraceID),
		zap.String("module", event.Module),
		zap.String("actor", event.Actor),
		zap.String("detail", string(detailJSON)),
		zap.String("created_at", event.CreatedAt.String()),
	}

	// 安全告警用 WARN 级别，其余用 INFO
	if event.EventType == EventSafetyAlert {
		w.logger.Warn("生命流事件", fields...)
	} else {
		w.logger.Info("生命流事件", fields...)
	}

	return nil
}

// --- 便捷写入函数 ---

// WriteAgentAction 写入 Agent 动作事件
func WriteAgentAction(ctx context.Context, w Writer, aggregateID, module, actor string, detail map[string]any) error {
	return w.Write(ctx, Event{
		EventType:   EventAgentAction,
		AggregateID: aggregateID,
		Module:      module,
		Actor:       actor,
		Detail:      detail,
	})
}

// WriteAgentError 写入 Agent 异常事件
func WriteAgentError(ctx context.Context, w Writer, aggregateID, module, actor string, err error) error {
	return w.Write(ctx, Event{
		EventType:   EventAgentError,
		AggregateID: aggregateID,
		Module:      module,
		Actor:       actor,
		Detail: map[string]any{
			"error": err.Error(),
		},
	})
}

// WriteStateChange 写入状态变更事件
func WriteStateChange(ctx context.Context, w Writer, aggregateID, module, from, to string) error {
	return w.Write(ctx, Event{
		EventType:   EventStateChange,
		AggregateID: aggregateID,
		Module:      module,
		Actor:       "system",
		Detail: map[string]any{
			"from": from,
			"to":   to,
		},
	})
}

// WriteToolCall 写入工具调用事件
func WriteToolCall(ctx context.Context, w Writer, aggregateID, module, toolName, actor string) error {
	return w.Write(ctx, Event{
		EventType:   EventToolCall,
		AggregateID: aggregateID,
		Module:      module,
		Actor:       actor,
		Detail: map[string]any{
			"tool": toolName,
		},
	})
}

// WriteSafetyAlert 写入安全告警事件
func WriteSafetyAlert(ctx context.Context, w Writer, aggregateID, module, alertType, reason string) error {
	return w.Write(ctx, Event{
		EventType:   EventSafetyAlert,
		AggregateID: aggregateID,
		Module:      module,
		Actor:       "system",
		Detail: map[string]any{
			"alert_type": alertType,
			"reason":     reason,
		},
	})
}
