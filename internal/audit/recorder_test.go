package audit

import (
	"context"
	"testing"
)

func TestNewLogRecorder(t *testing.T) {
	rec := NewLogRecorder()
	if rec == nil {
		t.Fatal("NewLogRecorder should not return nil")
	}
}

func TestRecordGeneratesID(t *testing.T) {
	rec := NewLogRecorder()

	err := rec.Record(context.Background(), Entry{
		Action:   ActionOrderCreate,
		Level:    LevelInfo,
		Module:   "dogwalk",
		UserID:   "user_001",
		EntityID: "order_001",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRecordWithPresetID(t *testing.T) {
	rec := NewLogRecorder()

	err := rec.Record(context.Background(), Entry{
		ID:       "audit_custom_001",
		Action:   ActionMedicalShare,
		Level:    LevelWarn,
		Module:   "medical",
		UserID:   "user_001",
		EntityID: "ep_001",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildDetail(t *testing.T) {
	detail := BuildDetail(
		"key1", "value1",
		"key2", 42,
		"key3", true,
	)

	if detail["key1"] != "value1" {
		t.Errorf("key1 = %v, want value1", detail["key1"])
	}
	if detail["key2"] != 42 {
		t.Errorf("key2 = %v, want 42", detail["key2"])
	}
	if detail["key3"] != true {
		t.Errorf("key3 = %v, want true", detail["key3"])
	}
}

func TestBuildDetailEmpty(t *testing.T) {
	detail := BuildDetail()
	if len(detail) != 0 {
		t.Errorf("detail len = %d, want 0", len(detail))
	}
}

func TestEntryString(t *testing.T) {
	e := Entry{
		Level:    LevelInfo,
		Module:   "medical",
		Action:   ActionMedicalShare,
		UserID:   "user_001",
		EntityID: "ep_001",
	}

	s := e.String()
	if s != "[INFO] medical/medical.share user=user_001 entity=ep_001" {
		t.Errorf("String() = %q", s)
	}
}

func TestActionConstants(t *testing.T) {
	actions := []Action{
		ActionMedicalShare, ActionMedicalAuth,
		ActionOrderCreate, ActionOrderPay, ActionOrderRefund,
		ActionThirdPartyCall, ActionToolExecute,
		ActionPolicyDeny, ActionLLMDegraded,
		ActionSagaCompensate, ActionHITLTrigger,
	}
	if len(actions) != 11 {
		t.Errorf("Action count = %d, want 11", len(actions))
	}
}

func TestLevelConstants(t *testing.T) {
	levels := []Level{LevelInfo, LevelWarn, LevelError}
	if len(levels) != 3 {
		t.Errorf("Level count = %d, want 3", len(levels))
	}
}
