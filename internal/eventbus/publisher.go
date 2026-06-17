package eventbus

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/patch-pet/patch-pet/pkg/logger"
)

// Publisher 事件发布器接口
type Publisher interface {
	// Publish 发布事件到对应 Topic
	Publish(ctx context.Context, event *Event) error
	// Close 关闭发布器
	Close() error
}

// LogPublisher 基于日志的事件发布器（开发/测试环境）
// 生产环境应替换为 Kafka 生产者
type LogPublisher struct{}

// NewLogPublisher 创建日志事件发布器
func NewLogPublisher() *LogPublisher {
	return &LogPublisher{}
}

// Publish 发布事件（写入结构化日志）
func (p *LogPublisher) Publish(ctx context.Context, event *Event) error {
	topic := event.GetTopic()

	data, err := event.ToJSON()
	if err != nil {
		return fmt.Errorf("序列化事件失败: %w", err)
	}

	logger.Info("事件发布",
		zap.String("event_id", event.EventID),
		zap.String("event_type", string(event.EventType)),
		zap.String("aggregate_id", event.AggregateID),
		zap.String("topic", string(topic)),
		zap.String("payload", string(data)),
	)

	return nil
}

// Close 关闭发布器
func (p *LogPublisher) Close() error {
	return nil
}

// RetryPublisher 带重试的事件发布器
// 重试 3 次后转入死信队列
type RetryPublisher struct {
	inner     Publisher
	maxRetry  int
	dlqPublisher Publisher
}

// NewRetryPublisher 创建带重试的发布器
func NewRetryPublisher(inner Publisher, dlqPublisher Publisher, maxRetry int) *RetryPublisher {
	return &RetryPublisher{
		inner:        inner,
		maxRetry:     maxRetry,
		dlqPublisher: dlqPublisher,
	}
}

// Publish 发布事件（带重试 + DLQ 兜底）
func (p *RetryPublisher) Publish(ctx context.Context, event *Event) error {
	var lastErr error

	for i := 0; i < p.maxRetry; i++ {
		if err := p.inner.Publish(ctx, event); err != nil {
			lastErr = err
			logger.Warn("事件发布失败，准备重试",
				zap.String("event_id", event.EventID),
				zap.Int("attempt", i+1),
				zap.Int("max_retry", p.maxRetry),
				zap.Error(err),
			)
			continue
		}
		return nil
	}

	// 重试耗尽，转入 DLQ
	logger.Error("事件发布重试耗尽，转入死信队列",
		zap.String("event_id", event.EventID),
		zap.String("event_type", string(event.EventType)),
		zap.Error(lastErr),
	)

	if p.dlqPublisher != nil {
		// DLQ 事件使用 DLQ Topic
		dlqEvent := &Event{
			EventID:     event.EventID,
			EventType:   event.EventType,
			AggregateID: event.AggregateID,
			Timestamp:   event.Timestamp,
			Payload:     event.Payload,
		}
		if dlqErr := p.dlqPublisher.Publish(ctx, dlqEvent); dlqErr != nil {
			logger.Error("死信队列写入失败",
				zap.String("event_id", event.EventID),
				zap.Error(dlqErr),
			)
		}
	}

	return fmt.Errorf("事件发布失败（已重试 %d 次）: %w", p.maxRetry, lastErr)
}

// Close 关闭发布器
func (p *RetryPublisher) Close() error {
	if err := p.inner.Close(); err != nil {
		return err
	}
	if p.dlqPublisher != nil {
		return p.dlqPublisher.Close()
	}
	return nil
}

// --- 便捷发布函数 ---

// PublishMedicalTaskCreated 发布医疗任务创建事件
func PublishMedicalTaskCreated(ctx context.Context, pub Publisher, taskID, episodeID string) error {
	event, err := NewEvent(EventMedicalTaskCreated, taskID, map[string]any{
		"task_id":    taskID,
		"episode_id": episodeID,
	})
	if err != nil {
		return err
	}
	return pub.Publish(ctx, event)
}

// PublishMedicalTaskCompleted 发布医疗任务完成事件
func PublishMedicalTaskCompleted(ctx context.Context, pub Publisher, taskID, episodeID string) error {
	event, err := NewEvent(EventMedicalTaskCompleted, taskID, map[string]any{
		"task_id":    taskID,
		"episode_id": episodeID,
	})
	if err != nil {
		return err
	}
	return pub.Publish(ctx, event)
}

// PublishMedicalSummaryGenerated 发布摘要生成事件
func PublishMedicalSummaryGenerated(ctx context.Context, pub Publisher, summaryID, episodeID string) error {
	event, err := NewEvent(EventMedicalSummaryGenerated, summaryID, map[string]any{
		"summary_id": summaryID,
		"episode_id": episodeID,
	})
	if err != nil {
		return err
	}
	return pub.Publish(ctx, event)
}

// PublishMedicalSummaryShared 发布摘要推送事件
func PublishMedicalSummaryShared(ctx context.Context, pub Publisher, summaryID, episodeID, clinicID string) error {
	event, err := NewEvent(EventMedicalSummaryShared, summaryID, map[string]any{
		"summary_id": summaryID,
		"episode_id": episodeID,
		"clinic_id":  clinicID,
	})
	if err != nil {
		return err
	}
	return pub.Publish(ctx, event)
}

// PublishDogwalkOpportunityCreated 发布代遛需求创建事件
func PublishDogwalkOpportunityCreated(ctx context.Context, pub Publisher, oppID, petID string) error {
	event, err := NewEvent(EventDogwalkOpportunityCreated, oppID, map[string]any{
		"opportunity_id": oppID,
		"pet_id":         petID,
	})
	if err != nil {
		return err
	}
	return pub.Publish(ctx, event)
}

// PublishDogwalkOrderPaid 发布代遛订单支付事件
func PublishDogwalkOrderPaid(ctx context.Context, pub Publisher, orderID, paymentOrderID string) error {
	event, err := NewEvent(EventDogwalkOrderPaid, orderID, map[string]any{
		"order_id":         orderID,
		"payment_order_id": paymentOrderID,
	})
	if err != nil {
		return err
	}
	return pub.Publish(ctx, event)
}

// PublishDogwalkOrderBooked 发布代遛订单预约事件
func PublishDogwalkOrderBooked(ctx context.Context, pub Publisher, orderID, vendorID string) error {
	event, err := NewEvent(EventDogwalkOrderBooked, orderID, map[string]any{
		"order_id":  orderID,
		"vendor_id": vendorID,
	})
	if err != nil {
		return err
	}
	return pub.Publish(ctx, event)
}

// PublishDogwalkOrderCompleted 发布代遛订单完成事件
func PublishDogwalkOrderCompleted(ctx context.Context, pub Publisher, orderID string) error {
	event, err := NewEvent(EventDogwalkOrderCompleted, orderID, map[string]any{
		"order_id": orderID,
	})
	if err != nil {
		return err
	}
	return pub.Publish(ctx, event)
}
