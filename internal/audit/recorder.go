package audit

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/patch-pet/patch-pet/internal/middleware"
	"github.com/patch-pet/patch-pet/pkg/logger"
	"github.com/patch-pet/patch-pet/pkg/types"
	"github.com/patch-pet/patch-pet/pkg/utils"
)

// LogRecorder 基于结构化日志的审计记录器
// 写入 zap JSON 日志，ERROR 级别必带 trace_id
type LogRecorder struct {
	logger *zap.Logger
}

// NewLogRecorder 创建日志审计记录器
func NewLogRecorder() *LogRecorder {
	return &LogRecorder{
		logger: logger.GetLogger().With(zap.String("component", "audit")),
	}
}

// Record 写入审计日志
func (r *LogRecorder) Record(ctx context.Context, entry Entry) error {
	// 自动生成 ID
	if entry.ID == "" {
		entry.ID = utils.GenerateULID(types.IDPrefix("audit"))
	}

	// 自动注入 trace_id
	if entry.TraceID == "" {
		entry.TraceID = middleware.GetTraceID(ctx)
	}

	// 自动注入时间戳
	if entry.CreatedAt.Time.IsZero() {
		entry.CreatedAt = types.NowCST()
	}

	// 序列化 detail
	detailJSON, _ := json.Marshal(entry.Detail)

	// 根据级别写入日志
	fields := []zap.Field{
		zap.String("audit_id", entry.ID),
		zap.String("trace_id", entry.TraceID),
		zap.String("action", string(entry.Action)),
		zap.String("level", string(entry.Level)),
		zap.String("module", entry.Module),
		zap.String("user_id", entry.UserID),
		zap.String("entity_id", entry.EntityID),
		zap.String("detail", string(detailJSON)),
		zap.String("created_at", entry.CreatedAt.String()),
	}

	switch entry.Level {
	case LevelError:
		r.logger.Error("审计日志", fields...)
	case LevelWarn:
		r.logger.Warn("审计日志", fields...)
	default:
		r.logger.Info("审计日志", fields...)
	}

	return nil
}

// RecordWithEntity 记录带实体信息的审计日志
func RecordWithEntity(ctx context.Context, recorder Recorder, action Action, level Level, module, userID, entityID string, detail map[string]any) error {
	return recorder.Record(ctx, Entry{
		Action:   action,
		Level:    level,
		Module:   module,
		UserID:   userID,
		EntityID: entityID,
		Detail:   detail,
	})
}

// RecordMedicalShare 记录医疗数据外发（P0 级审计）
func RecordMedicalShare(ctx context.Context, recorder Recorder, userID, episodeID, recipientType, recipientID string) error {
	return recorder.Record(ctx, Entry{
		Action:   ActionMedicalShare,
		Level:    LevelWarn, // 医疗外发必须 WARN 级别
		Module:   "medical",
		UserID:   userID,
		EntityID: episodeID,
		Detail: map[string]any{
			"recipient_type": recipientType,
			"recipient_id":   recipientID,
		},
	})
}

// RecordOrderPayment 记录支付操作（P0 级审计）
func RecordOrderPayment(ctx context.Context, recorder Recorder, userID, orderID, paymentOrderID string, amountCents int) error {
	return recorder.Record(ctx, Entry{
		Action:   ActionOrderPay,
		Level:    LevelInfo,
		Module:   "dogwalk",
		UserID:   userID,
		EntityID: orderID,
		Detail: map[string]any{
			"payment_order_id": paymentOrderID,
			"amount_cents":     amountCents,
		},
	})
}

// RecordPolicyDeny 记录策略拒绝
func RecordPolicyDeny(ctx context.Context, recorder Recorder, module, toolName, permission, reason string) error {
	return recorder.Record(ctx, Entry{
		Action: ActionPolicyDeny,
		Level:  LevelWarn,
		Module: module,
		Detail: map[string]any{
			"tool":       toolName,
			"permission": permission,
			"reason":     reason,
		},
	})
}

// BuildDetail 构建审计详情（辅助函数）
func BuildDetail(kv ...interface{}) map[string]any {
	detail := make(map[string]any, len(kv)/2)
	for i := 0; i < len(kv)-1; i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			continue
		}
		detail[key] = kv[i+1]
	}
	return detail
}

// String 返回审计条目的可读表示（用于调试）
func (e Entry) String() string {
	return fmt.Sprintf("[%s] %s/%s user=%s entity=%s", e.Level, e.Module, e.Action, e.UserID, e.EntityID)
}
