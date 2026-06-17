package eventbus

import (
	"context"
	"errors"
	"testing"
)

func TestAIEventTypeConstants(t *testing.T) {
	types := AIEventTypes()
	if len(types) != 6 {
		t.Errorf("AIEventType count = %d, want 6", len(types))
	}
}

func TestAIMessageStruct(t *testing.T) {
	msg := AIMessage{
		MessageID: "msg_123",
		EventType: AIEventRAGUpdate,
		Source:    "go-service",
		Payload:   []byte(`{"key":"value"}`),
	}
	if msg.Source != "go-service" {
		t.Errorf("Source = %s, want go-service", msg.Source)
	}
}

func TestRAGUpdatePayload(t *testing.T) {
	payload := RAGUpdatePayload{
		DocumentID:   "doc_1",
		Action:       "add",
		Content:      "test content",
		ChunkSize:    512,
		ChunkOverlap: 64,
	}
	if payload.ChunkSize != 512 {
		t.Errorf("ChunkSize = %d, want 512", payload.ChunkSize)
	}
}

func TestEvalReportPayload(t *testing.T) {
	payload := EvalReportPayload{
		ReportID:   "rpt_1",
		EvalType:   "medical",
		TotalCases: 1000,
		PassRate:   0.95,
		P0HitRate:  1.0,
	}
	if payload.P0HitRate != 1.0 {
		t.Errorf("P0HitRate = %f, want 1.0", payload.P0HitRate)
	}
}

func TestPromptUpdatePayload(t *testing.T) {
	payload := PromptUpdatePayload{
		PromptID: "prompt_1",
		Version:  "v1.2",
		Action:   "deploy",
	}
	if payload.Action != "deploy" {
		t.Errorf("Action = %s, want deploy", payload.Action)
	}
}

type mockAI Publisher

func TestAIBridgePublishRAGUpdate(t *testing.T) {
	pub := NewLogPublisher()
	bridge := NewAIBridge(pub, 3)

	err := bridge.PublishRAGUpdate(context.Background(), RAGUpdatePayload{
		DocumentID: "doc_1",
		Action:     "add",
		Content:    "test",
	})
	if err != nil {
		t.Fatalf("PublishRAGUpdate failed: %v", err)
	}
}

func TestAIBridgePublishEvalReport(t *testing.T) {
	pub := NewLogPublisher()
	bridge := NewAIBridge(pub, 3)

	err := bridge.PublishEvalReport(context.Background(), EvalReportPayload{
		ReportID:   "rpt_1",
		EvalType:   "medical",
		TotalCases: 1000,
		P0HitRate:  1.0,
	})
	if err != nil {
		t.Fatalf("PublishEvalReport failed: %v", err)
	}
}

func TestAIBridgePublishPromptUpdate(t *testing.T) {
	pub := NewLogPublisher()
	bridge := NewAIBridge(pub, 3)

	err := bridge.PublishPromptUpdate(context.Background(), PromptUpdatePayload{
		PromptID: "prompt_1",
		Version:  "v1.0",
		Action:   "deploy",
	})
	if err != nil {
		t.Fatalf("PublishPromptUpdate failed: %v", err)
	}
}

type failPublisher struct {
	failCount int
	calls     int
}

func (f *failPublisher) Publish(ctx context.Context, event *Event) error {
	f.calls++
	if f.calls <= f.failCount {
		return errors.New("publish failed")
	}
	return nil
}

func (f *failPublisher) Close() error { return nil }

func TestAIBridgeRetryAndDLQ(t *testing.T) {
	pub := &failPublisher{failCount: 99} // Always fail
	bridge := NewAIBridge(pub, 3)

	err := bridge.PublishRAGUpdate(context.Background(), RAGUpdatePayload{
		DocumentID: "doc_1",
		Action:     "add",
	})
	if err == nil {
		t.Fatal("Should fail after max retries")
	}
	if pub.calls != 4 { // 1 initial + 3 retries
		t.Errorf("calls = %d, want 4", pub.calls)
	}
}

func TestAIBridgeRetrySuccess(t *testing.T) {
	pub := &failPublisher{failCount: 2} // Fail first 2, succeed on 3rd
	bridge := NewAIBridge(pub, 3)

	err := bridge.PublishRAGUpdate(context.Background(), RAGUpdatePayload{
		DocumentID: "doc_1",
		Action:     "add",
	})
	if err != nil {
		t.Fatalf("Should succeed on retry: %v", err)
	}
}
