package eventbus

import (
	"encoding/json"
	"testing"
)

func TestTopicConstants(t *testing.T) {
	topics := []Topic{
		TopicMedicalEvent, TopicDogwalkEvent, TopicAIOfflineTask, TopicDLQ,
	}
	if len(topics) != 4 {
		t.Errorf("Topic count = %d, want 4", len(topics))
	}
}

func TestEventTypeConstants(t *testing.T) {
	medicalEvents := []EventType{
		EventMedicalTaskCreated, EventMedicalTaskCompleted,
		EventMedicalSummaryGenerated, EventMedicalSummaryShared,
	}
	if len(medicalEvents) != 4 {
		t.Errorf("Medical EventType count = %d, want 4", len(medicalEvents))
	}

	dogwalkEvents := []EventType{
		EventDogwalkOpportunityCreated, EventDogwalkPlanCreated,
		EventDogwalkOrderPaid, EventDogwalkOrderBooked, EventDogwalkOrderCompleted,
	}
	if len(dogwalkEvents) != 5 {
		t.Errorf("Dogwalk EventType count = %d, want 5", len(dogwalkEvents))
	}
}

func TestNewEvent(t *testing.T) {
	event, err := NewEvent(EventMedicalTaskCreated, "task_123", map[string]any{
		"task_id": "task_123",
	})
	if err != nil {
		t.Fatalf("NewEvent failed: %v", err)
	}

	if event.EventID == "" {
		t.Error("EventID should not be empty")
	}
	if event.EventType != EventMedicalTaskCreated {
		t.Errorf("EventType = %s, want %s", event.EventType, EventMedicalTaskCreated)
	}
	if event.AggregateID != "task_123" {
		t.Errorf("AggregateID = %s, want task_123", event.AggregateID)
	}
	if event.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
	if len(event.Payload) == 0 {
		t.Error("Payload should not be empty")
	}
}

func TestEventToJSON(t *testing.T) {
	event, _ := NewEvent(EventMedicalTaskCreated, "task_1", map[string]any{"key": "value"})
	data, err := event.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed["event_type"] != string(EventMedicalTaskCreated) {
		t.Errorf("event_type = %v, want %s", parsed["event_type"], EventMedicalTaskCreated)
	}
}

func TestEventFromJSON(t *testing.T) {
	event, _ := NewEvent(EventDogwalkOrderPaid, "order_1", map[string]any{"amount": 100})
	data, _ := event.ToJSON()

	parsed, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if parsed.EventType != EventDogwalkOrderPaid {
		t.Errorf("EventType = %s, want %s", parsed.EventType, EventDogwalkOrderPaid)
	}
	if parsed.AggregateID != "order_1" {
		t.Errorf("AggregateID = %s, want order_1", parsed.AggregateID)
	}
}

func TestEventGetTopicMedical(t *testing.T) {
	tests := []struct {
		eventType EventType
		want      Topic
	}{
		{EventMedicalTaskCreated, TopicMedicalEvent},
		{EventMedicalTaskCompleted, TopicMedicalEvent},
		{EventMedicalSummaryGenerated, TopicMedicalEvent},
		{EventMedicalSummaryShared, TopicMedicalEvent},
	}

	for _, tt := range tests {
		event, _ := NewEvent(tt.eventType, "id", nil)
		if got := event.GetTopic(); got != tt.want {
			t.Errorf("GetTopic(%s) = %s, want %s", tt.eventType, got, tt.want)
		}
	}
}

func TestEventGetTopicDogwalk(t *testing.T) {
	tests := []struct {
		eventType EventType
		want      Topic
	}{
		{EventDogwalkOpportunityCreated, TopicDogwalkEvent},
		{EventDogwalkPlanCreated, TopicDogwalkEvent},
		{EventDogwalkOrderPaid, TopicDogwalkEvent},
		{EventDogwalkOrderBooked, TopicDogwalkEvent},
		{EventDogwalkOrderCompleted, TopicDogwalkEvent},
	}

	for _, tt := range tests {
		event, _ := NewEvent(tt.eventType, "id", nil)
		if got := event.GetTopic(); got != tt.want {
			t.Errorf("GetTopic(%s) = %s, want %s", tt.eventType, got, tt.want)
		}
	}
}

func TestEventGetTopicUnknownGoesToDLQ(t *testing.T) {
	event, _ := NewEvent(EventType("unknown.event"), "id", nil)
	if got := event.GetTopic(); got != TopicDLQ {
		t.Errorf("GetTopic(unknown) = %s, want %s", got, TopicDLQ)
	}
}
