package eventbus

import (
	"context"
	"errors"
	"testing"
)

func TestLogPublisherPublish(t *testing.T) {
	pub := NewLogPublisher()
	event, _ := NewEvent(EventMedicalTaskCreated, "task_1", map[string]any{"key": "value"})

	err := pub.Publish(context.Background(), event)
	if err != nil {
		t.Fatalf("LogPublisher.Publish failed: %v", err)
	}
}

func TestLogPublisherClose(t *testing.T) {
	pub := NewLogPublisher()
	if err := pub.Close(); err != nil {
		t.Fatalf("LogPublisher.Close failed: %v", err)
	}
}

type mockPublisher struct {
	callCount int
	failUntil int
}

func (m *mockPublisher) Publish(ctx context.Context, event *Event) error {
	m.callCount++
	if m.callCount <= m.failUntil {
		return errors.New("publish failed")
	}
	return nil
}

func (m *mockPublisher) Close() error {
	return nil
}

func TestRetryPublisherSuccess(t *testing.T) {
	inner := &mockPublisher{failUntil: 0}
	dlq := &mockPublisher{failUntil: 0}
	pub := NewRetryPublisher(inner, dlq, 3)

	event, _ := NewEvent(EventMedicalTaskCreated, "task_1", nil)
	err := pub.Publish(context.Background(), event)
	if err != nil {
		t.Fatalf("RetryPublisher should succeed: %v", err)
	}
	if inner.callCount != 1 {
		t.Errorf("Inner callCount = %d, want 1", inner.callCount)
	}
}

func TestRetryPublisherRetries(t *testing.T) {
	inner := &mockPublisher{failUntil: 2} // Fail first 2 times
	dlq := &mockPublisher{failUntil: 0}
	pub := NewRetryPublisher(inner, dlq, 3)

	event, _ := NewEvent(EventMedicalTaskCreated, "task_1", nil)
	err := pub.Publish(context.Background(), event)
	if err != nil {
		t.Fatalf("RetryPublisher should succeed on 3rd try: %v", err)
	}
	if inner.callCount != 3 {
		t.Errorf("Inner callCount = %d, want 3", inner.callCount)
	}
}

func TestRetryPublisherDLQ(t *testing.T) {
	inner := &mockPublisher{failUntil: 99} // Always fail
	dlq := &mockPublisher{failUntil: 0}
	pub := NewRetryPublisher(inner, dlq, 3)

	event, _ := NewEvent(EventMedicalTaskCreated, "task_1", nil)
	err := pub.Publish(context.Background(), event)
	if err == nil {
		t.Fatal("RetryPublisher should fail after max retries")
	}
	if inner.callCount != 3 {
		t.Errorf("Inner callCount = %d, want 3", inner.callCount)
	}
	if dlq.callCount != 1 {
		t.Errorf("DLQ callCount = %d, want 1", dlq.callCount)
	}
}

func TestConveniencePublisherFunctions(t *testing.T) {
	pub := NewLogPublisher()
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"PublishMedicalTaskCreated", func() error {
			return PublishMedicalTaskCreated(ctx, pub, "task_1", "ep_1")
		}},
		{"PublishMedicalTaskCompleted", func() error {
			return PublishMedicalTaskCompleted(ctx, pub, "task_1", "ep_1")
		}},
		{"PublishMedicalSummaryGenerated", func() error {
			return PublishMedicalSummaryGenerated(ctx, pub, "sum_1", "ep_1")
		}},
		{"PublishMedicalSummaryShared", func() error {
			return PublishMedicalSummaryShared(ctx, pub, "sum_1", "ep_1", "clinic_1")
		}},
		{"PublishDogwalkOpportunityCreated", func() error {
			return PublishDogwalkOpportunityCreated(ctx, pub, "opp_1", "pet_1")
		}},
		{"PublishDogwalkOrderPaid", func() error {
			return PublishDogwalkOrderPaid(ctx, pub, "order_1", "pay_1")
		}},
		{"PublishDogwalkOrderBooked", func() error {
			return PublishDogwalkOrderBooked(ctx, pub, "order_1", "vendor_1")
		}},
		{"PublishDogwalkOrderCompleted", func() error {
			return PublishDogwalkOrderCompleted(ctx, pub, "order_1")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); err != nil {
				t.Errorf("%s failed: %v", tt.name, err)
			}
		})
	}
}
