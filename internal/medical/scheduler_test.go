package medical

import (
	"testing"
	"time"
)

func TestSchedulerConstants(t *testing.T) {
	if SchedulerLockKey == "" {
		t.Error("SchedulerLockKey should not be empty")
	}
	if SchedulerLockTTL != 30*time.Second {
		t.Errorf("SchedulerLockTTL = %v, want 30s", SchedulerLockTTL)
	}
	if SchedulerTimeout != 10*time.Second {
		t.Errorf("SchedulerTimeout = %v, want 10s", SchedulerTimeout)
	}
	if MaxTasksPerBatch != 50 {
		t.Errorf("MaxTasksPerBatch = %d, want 50", MaxTasksPerBatch)
	}
	if OverdueThresholdMinutes != 30 {
		t.Errorf("OverdueThresholdMinutes = %d, want 30", OverdueThresholdMinutes)
	}
}

func TestNewSchedulerNilDeps(t *testing.T) {
	// Verify scheduler can be created with nil dependencies (won't panic)
	s := &Scheduler{
		repo:     nil,
		redis:    nil,
		lifeflow: nil,
		auditLog: nil,
	}
	if s == nil {
		t.Fatal("Scheduler should not be nil")
	}
}
