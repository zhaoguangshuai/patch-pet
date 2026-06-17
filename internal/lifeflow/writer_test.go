package lifeflow

import (
	"context"
	"errors"
	"testing"
)

func TestNewLogWriter(t *testing.T) {
	w := NewLogWriter()
	if w == nil {
		t.Fatal("NewLogWriter should not return nil")
	}
}

func TestWriteGeneratesID(t *testing.T) {
	w := NewLogWriter()

	err := w.Write(context.Background(), Event{
		EventType:   EventAgentAction,
		AggregateID: "ep_001",
		Module:      "medical",
		Actor:       "agent",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteWithPresetID(t *testing.T) {
	w := NewLogWriter()

	err := w.Write(context.Background(), Event{
		ID:          "evt_custom_001",
		EventType:   EventStateChange,
		AggregateID: "ep_001",
		Module:      "medical",
		Actor:       "system",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteSafetyAlertUsesWarnLevel(t *testing.T) {
	w := NewLogWriter()

	err := w.Write(context.Background(), Event{
		EventType:   EventSafetyAlert,
		AggregateID: "ep_001",
		Module:      "medical",
		Actor:       "system",
		Detail: map[string]any{
			"alert_type": "diagnosis_blocked",
			"reason":     "LLM 输出包含诊断内容",
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEventTypeConstants(t *testing.T) {
	types := []EventType{
		EventAgentAction, EventAgentError, EventStateChange,
		EventToolCall, EventSafetyAlert, EventUserConfirm, EventSystemNotify,
	}
	if len(types) != 7 {
		t.Errorf("EventType count = %d, want 7", len(types))
	}
}

func TestWriterInterface(t *testing.T) {
	var w Writer = NewLogWriter()
	if w == nil {
		t.Error("LogWriter should implement Writer interface")
	}
}

func TestWriteAgentAction(t *testing.T) {
	w := NewLogWriter()

	err := WriteAgentAction(context.Background(), w, "ep_001", "medical", "agent", map[string]any{
		"tool": "get_medical_episode",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteAgentError(t *testing.T) {
	w := NewLogWriter()

	err := WriteAgentError(context.Background(), w, "ep_001", "medical", "agent", errors.New("tool execution failed"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteStateChange(t *testing.T) {
	w := NewLogWriter()

	err := WriteStateChange(context.Background(), w, "ep_001", "medical", "Created", "Active")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteToolCall(t *testing.T) {
	w := NewLogWriter()

	err := WriteToolCall(context.Background(), w, "ep_001", "medical", "get_episode", "agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteSafetyAlert(t *testing.T) {
	w := NewLogWriter()

	err := WriteSafetyAlert(context.Background(), w, "ep_001", "medical", "diagnosis_blocked", "LLM 输出包含诊断内容")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
