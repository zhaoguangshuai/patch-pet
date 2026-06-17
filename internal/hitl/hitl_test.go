package hitl

import (
	"context"
	"errors"
	"testing"

	"github.com/patch-pet/patch-pet/internal/audit"
)

type mockAudit struct {
	entries []audit.Entry
}

func (m *mockAudit) Record(ctx context.Context, entry audit.Entry) error {
	m.entries = append(m.entries, entry)
	return nil
}

func TestFireCreatesWorkOrder(t *testing.T) {
	ma := &mockAudit{}
	tr := NewTrigger(ma)

	order, err := tr.Fire(context.Background(), TriggerP0Risk, "medical", "P0 风险操作", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.ID == "" {
		t.Error("order ID should not be empty")
	}
	if order.Status != StatusOpen {
		t.Errorf("status = %s, want open", order.Status)
	}
	if order.Priority != "P0" {
		t.Errorf("priority = %s, want P0", order.Priority)
	}
	if len(ma.entries) != 1 {
		t.Errorf("audit entries = %d, want 1", len(ma.entries))
	}
}

func TestFireAgentConsecutiveFail(t *testing.T) {
	tr := NewTrigger(nil)

	// First 2 failures: no HITL
	for i := 0; i < 2; i++ {
		order, err := tr.RecordAgentFailure(context.Background(), "medical", errors.New("fail"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if order != nil {
			t.Error("should not trigger HITL before 3 failures")
		}
	}

	// Third failure: trigger HITL
	order, err := tr.RecordAgentFailure(context.Background(), "medical", errors.New("fail"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order == nil {
		t.Fatal("should trigger HITL on 3rd failure")
	}
	if order.TriggerType != TriggerAgentConsecutiveFail {
		t.Errorf("trigger type = %s, want agent_consecutive_fail", order.TriggerType)
	}
}

func TestResetAgentFailures(t *testing.T) {
	tr := NewTrigger(nil)

	tr.RecordAgentFailure(context.Background(), "medical", errors.New("fail"))
	tr.RecordAgentFailure(context.Background(), "medical", errors.New("fail"))
	tr.ResetAgentFailures("medical")

	// Should not trigger after reset
	for i := 0; i < 2; i++ {
		order, _ := tr.RecordAgentFailure(context.Background(), "medical", errors.New("fail"))
		if order != nil {
			t.Error("should not trigger after reset")
		}
	}
}

func TestGetOrder(t *testing.T) {
	tr := NewTrigger(nil)
	order, _ := tr.Fire(context.Background(), TriggerUserComplaint, "support", "用户投诉", nil)

	found, ok := tr.GetOrder(order.ID)
	if !ok {
		t.Error("should find order")
	}
	if found.ID != order.ID {
		t.Errorf("ID mismatch: %s vs %s", found.ID, order.ID)
	}

	_, ok = tr.GetOrder("nonexistent")
	if ok {
		t.Error("should not find nonexistent order")
	}
}

func TestListOpenOrders(t *testing.T) {
	tr := NewTrigger(nil)
	tr.Fire(context.Background(), TriggerP0Risk, "medical", "test1", nil)
	tr.Fire(context.Background(), TriggerUserComplaint, "support", "test2", nil)

	orders := tr.ListOpenOrders()
	if len(orders) != 2 {
		t.Errorf("open orders = %d, want 2", len(orders))
	}
}

func TestUpdateStatus(t *testing.T) {
	tr := NewTrigger(nil)
	order, _ := tr.Fire(context.Background(), TriggerP0Risk, "medical", "test", nil)

	err := tr.UpdateStatus(order.ID, StatusInProgress, "admin", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, _ := tr.GetOrder(order.ID)
	if found.Status != StatusInProgress {
		t.Errorf("status = %s, want in_progress", found.Status)
	}
	if found.Assignee != "admin" {
		t.Errorf("assignee = %s, want admin", found.Assignee)
	}
}

func TestUpdateStatusNotFound(t *testing.T) {
	tr := NewTrigger(nil)
	err := tr.UpdateStatus("nonexistent", StatusResolved, "", "fixed")
	if err == nil {
		t.Error("expected error for nonexistent order")
	}
}

func TestPriorityMapping(t *testing.T) {
	tr := NewTrigger(nil)

	tests := []struct {
		trigger  TriggerType
		expected string
	}{
		{TriggerP0Risk, "P0"},
		{TriggerSagaFailure, "P0"},
		{TriggerAgentConsecutiveFail, "P1"},
		{TriggerPermissionConflict, "P1"},
		{TriggerUserComplaint, "P2"},
	}

	for _, tt := range tests {
		order, _ := tr.Fire(context.Background(), tt.trigger, "test", "desc", nil)
		if string(order.Priority) != tt.expected {
			t.Errorf("trigger %s: priority = %s, want %s", tt.trigger, order.Priority, tt.expected)
		}
	}
}
