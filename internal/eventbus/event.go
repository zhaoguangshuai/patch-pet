// Package eventbus Kafka 事件总线
// 标准事件结构体 + Topic 划分 + 事件发布器
// 3 次失败转 biz-dlq 死信队列
package eventbus

import (
	"encoding/json"
	"time"

	"github.com/patch-pet/patch-pet/pkg/types"
	"github.com/patch-pet/patch-pet/pkg/utils"
)

// Topic Kafka Topic 枚举
type Topic string

const (
	TopicMedicalEvent  Topic = "biz-medical-event"  // 医疗业务事件
	TopicDogwalkEvent  Topic = "biz-dogwalk-event"  // 代遛业务事件
	TopicAIOfflineTask Topic = "ai-offline-task"     // AI 离线任务
	TopicDLQ           Topic = "biz-dlq"             // 死信队列
)

// EventType 业务事件类型枚举
type EventType string

const (
	// 医疗事件
	EventMedicalTaskCreated   EventType = "medical.task.created"
	EventMedicalTaskCompleted EventType = "medical.task.completed"
	EventMedicalSummaryGenerated EventType = "medical.summary.generated"
	EventMedicalSummaryShared    EventType = "medical.summary.shared"

	// 代遛事件
	EventDogwalkOpportunityCreated EventType = "dogwalk.opportunity.created"
	EventDogwalkPlanCreated        EventType = "dogwalk.plan.created"
	EventDogwalkOrderPaid          EventType = "dogwalk.order.paid"
	EventDogwalkOrderBooked        EventType = "dogwalk.order.booked"
	EventDogwalkOrderCompleted     EventType = "dogwalk.order.completed"
)

// Event 标准业务事件结构体
type Event struct {
	EventID     string          `json:"event_id"`     // 事件唯一 ID（ULID）
	EventType   EventType       `json:"event_type"`   // 事件类型
	AggregateID string          `json:"aggregate_id"` // 聚合根 ID
	Timestamp   time.Time       `json:"timestamp"`    // 事件时间（UTC）
	Payload     json.RawMessage `json:"payload"`      // 事件数据
}

// NewEvent 创建标准事件
func NewEvent(eventType EventType, aggregateID string, payload any) (*Event, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &Event{
		EventID:     utils.GenerateULID(types.IDPrefix("evt")),
		EventType:   eventType,
		AggregateID: aggregateID,
		Timestamp:   time.Now().UTC(),
		Payload:     payloadBytes,
	}, nil
}

// ToJSON 序列化为 JSON
func (e *Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// FromJSON 从 JSON 反序列化
func FromJSON(data []byte) (*Event, error) {
	var e Event
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// GetTopic 根据事件类型获取对应 Topic
func (e *Event) GetTopic() Topic {
	switch {
	case e.EventType == EventMedicalTaskCreated ||
		e.EventType == EventMedicalTaskCompleted ||
		e.EventType == EventMedicalSummaryGenerated ||
		e.EventType == EventMedicalSummaryShared:
		return TopicMedicalEvent
	case e.EventType == EventDogwalkOpportunityCreated ||
		e.EventType == EventDogwalkPlanCreated ||
		e.EventType == EventDogwalkOrderPaid ||
		e.EventType == EventDogwalkOrderBooked ||
		e.EventType == EventDogwalkOrderCompleted:
		return TopicDogwalkEvent
	default:
		return TopicDLQ
	}
}
