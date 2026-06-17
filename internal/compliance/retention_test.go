package compliance

import (
	"testing"
	"time"
)

func TestGetPolicyMedical(t *testing.T) {
	m := NewRetentionManager()
	p, err := m.GetPolicy(CategoryMedical)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.RetentionDays != 5*365 {
		t.Errorf("retention = %d, want %d", p.RetentionDays, 5*365)
	}
	if !p.EncryptOnStore {
		t.Error("medical should require encryption")
	}
}

func TestGetPolicyPayment(t *testing.T) {
	m := NewRetentionManager()
	p, err := m.GetPolicy(CategoryPayment)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.RetentionDays != 7*365 {
		t.Errorf("retention = %d, want %d", p.RetentionDays, 7*365)
	}
	if !p.TamperProof {
		t.Error("payment should be tamper-proof")
	}
}

func TestGetPolicyTrajectory(t *testing.T) {
	m := NewRetentionManager()
	p, err := m.GetPolicy(CategoryTrajectory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.RetentionDays != 180 {
		t.Errorf("retention = %d, want 180", p.RetentionDays)
	}
	if !p.DesensitizeOnExpire {
		t.Error("trajectory should desensitize on expire")
	}
}

func TestGetPolicyAudit(t *testing.T) {
	m := NewRetentionManager()
	p, err := m.GetPolicy(CategoryAudit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.ReadOnly {
		t.Error("audit should be read-only")
	}
	if !p.TamperProof {
		t.Error("audit should be tamper-proof")
	}
}

func TestGetPolicyUnknown(t *testing.T) {
	m := NewRetentionManager()
	_, err := m.GetPolicy("unknown")
	if err == nil {
		t.Error("expected error for unknown category")
	}
}

func TestIsExpired(t *testing.T) {
	m := NewRetentionManager()

	// 6 年前的医疗数据应过期
	old := time.Now().AddDate(-6, 0, 0)
	expired, err := m.IsExpired(CategoryMedical, old)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !expired {
		t.Error("6-year-old medical data should be expired")
	}

	// 1 年前的医疗数据不应过期
	recent := time.Now().AddDate(-1, 0, 0)
	expired, err = m.IsExpired(CategoryMedical, recent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expired {
		t.Error("1-year-old medical data should not be expired")
	}
}

func TestShouldArchive(t *testing.T) {
	m := NewRetentionManager()

	// 100 天前的轨迹数据应归档（归档天数 90）
	old := time.Now().AddDate(0, 0, -100)
	archive, err := m.ShouldArchive(CategoryTrajectory, old)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !archive {
		t.Error("100-day-old trajectory should be archived")
	}

	// 会话数据无归档
	archive, err = m.ShouldArchive(CategorySession, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if archive {
		t.Error("session data should not archive")
	}
}

func TestNeedsEncryption(t *testing.T) {
	m := NewRetentionManager()

	enc, _ := m.NeedsEncryption(CategoryMedical)
	if !enc {
		t.Error("medical should need encryption")
	}

	enc, _ = m.NeedsEncryption(CategoryTrajectory)
	if enc {
		t.Error("trajectory should not need encryption")
	}
}

func TestUserDeletionPlan(t *testing.T) {
	m := NewRetentionManager()
	sop := NewUserDeletionSOP(m)
	plan := sop.GetDeletionPlan("user_001")

	if len(plan) != 6 {
		t.Errorf("plan steps = %d, want 6", len(plan))
	}
	if plan[0].DaysOffset != 0 {
		t.Errorf("step 1 offset = %d, want 0", plan[0].DaysOffset)
	}
	if plan[5].DaysOffset != 7 {
		t.Errorf("step 6 offset = %d, want 7", plan[5].DaysOffset)
	}
}
