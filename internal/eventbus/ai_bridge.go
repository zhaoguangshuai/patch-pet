// Package eventbus Go ↔ Python AI 服务 Kafka 桥接
// 异步 RAG 更新、评测上报；3 次失败转 biz-dlq
package eventbus

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/patch-pet/patch-pet/pkg/logger"
	"github.com/patch-pet/patch-pet/pkg/types"
	"github.com/patch-pet/patch-pet/pkg/utils"
)

// AIEventType AI 服务事件类型
type AIEventType string

const (
	AIEventRAGUpdate    AIEventType = "ai.rag.update"    // RAG 知识库更新
	AIEventRAGIndex     AIEventType = "ai.rag.index"     // RAG 文档索引
	AIEventEvalReport   AIEventType = "ai.eval.report"   // 评测报告上报
	AIEventEvalResult   AIEventType = "ai.eval.result"   // 评测结果
	AIEventPromptUpdate AIEventType = "ai.prompt.update" // Prompt 版本更新
	AIEventModelSwitch  AIEventType = "ai.model.switch"  // 模型切换通知
)

// AIMessage AI 服务 Kafka 消息
type AIMessage struct {
	MessageID string          `json:"message_id"` // ULID
	EventType AIEventType     `json:"event_type"`
	Source    string          `json:"source"` // go-service / ai-service
	Payload   json.RawMessage `json:"payload"`
}

// RAGUpdatePayload RAG 更新载荷
type RAGUpdatePayload struct {
	DocumentID   string `json:"document_id"`
	Action       string `json:"action"` // add / update / delete
	Content      string `json:"content,omitempty"`
	ChunkSize    int    `json:"chunk_size,omitempty"`
	ChunkOverlap int    `json:"chunk_overlap,omitempty"`
}

// EvalReportPayload 评测报告载荷
type EvalReportPayload struct {
	ReportID   string         `json:"report_id"`
	EvalType   string         `json:"eval_type"` // medical / dogwalk
	TotalCases int            `json:"total_cases"`
	PassRate   float64        `json:"pass_rate"`
	P0HitRate  float64        `json:"p0_hit_rate"` // P0 红线拦截率
	Details    map[string]any `json:"details,omitempty"`
}

// PromptUpdatePayload Prompt 更新载荷
type PromptUpdatePayload struct {
	PromptID string `json:"prompt_id"`
	Version  string `json:"version"`
	Action   string `json:"action"` // deploy / rollback
}

// AIBridge Go ↔ Python AI 服务 Kafka 桥接器
type AIBridge struct {
	publisher Publisher
	maxRetry  int
}

// NewAIBridge 创建 AI 桥接器
func NewAIBridge(pub Publisher, maxRetry int) *AIBridge {
	return &AIBridge{
		publisher: pub,
		maxRetry:  maxRetry,
	}
}

// PublishRAGUpdate 发布 RAG 更新消息（Go → Python）
func (b *AIBridge) PublishRAGUpdate(ctx context.Context, payload RAGUpdatePayload) error {
	return b.publish(ctx, AIEventRAGUpdate, "go-service", payload)
}

// PublishRAGIndex 发布 RAG 索引消息（Go → Python）
func (b *AIBridge) PublishRAGIndex(ctx context.Context, documentID string) error {
	return b.publish(ctx, AIEventRAGIndex, "go-service", map[string]any{
		"document_id": documentID,
	})
}

// PublishEvalReport 发布评测报告（Python → Go，通过 Kafka 桥接）
func (b *AIBridge) PublishEvalReport(ctx context.Context, payload EvalReportPayload) error {
	return b.publish(ctx, AIEventEvalReport, "ai-service", payload)
}

// PublishPromptUpdate 发布 Prompt 更新通知（Go → Python）
func (b *AIBridge) PublishPromptUpdate(ctx context.Context, payload PromptUpdatePayload) error {
	return b.publish(ctx, AIEventPromptUpdate, "go-service", payload)
}

// publish 统一发布方法（3 次失败转 DLQ）
func (b *AIBridge) publish(ctx context.Context, eventType AIEventType, source string, payload any) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化 AI 消息失败: %w", err)
	}

	msg := AIMessage{
		MessageID: utils.GenerateULID(types.IDPrefix("evt")),
		EventType: eventType,
		Source:    source,
		Payload:   payloadBytes,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化 AIMessage 失败: %w", err)
	}

	event := &Event{
		EventID:     msg.MessageID,
		EventType:   EventType(eventType),
		AggregateID: "ai-service",
		Payload:     msgBytes,
	}

	var lastErr error
	for i := 0; i <= b.maxRetry; i++ {
		if err := b.publisher.Publish(ctx, event); err != nil {
			lastErr = err
			logger.Warn("AI 桥接消息发送失败",
				zap.String("event_type", string(eventType)),
				zap.Int("attempt", i+1),
				zap.Error(err),
			)
			continue
		}
		return nil
	}

	logger.Error("AI 桥接消息发送失败，转死信队列",
		zap.String("event_type", string(eventType)),
		zap.String("message_id", msg.MessageID),
	)
	return fmt.Errorf("AI 桥接消息发送失败（已重试 %d 次）: %w", b.maxRetry, lastErr)
}

// AIEventTypes 返回所有 AI 事件类型
func AIEventTypes() []AIEventType {
	return []AIEventType{
		AIEventRAGUpdate,
		AIEventRAGIndex,
		AIEventEvalReport,
		AIEventEvalResult,
		AIEventPromptUpdate,
		AIEventModelSwitch,
	}
}
