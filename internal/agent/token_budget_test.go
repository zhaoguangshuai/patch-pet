package agent

import (
	"testing"
)

func TestBudgetConfigDefaults(t *testing.T) {
	cfg := DefaultBudgetConfig()
	if cfg.DailyBudget != DefaultDailyBudget {
		t.Errorf("DailyBudget = %d, want %d", cfg.DailyBudget, DefaultDailyBudget)
	}
	if cfg.MonthlyBudget != DefaultMonthlyBudget {
		t.Errorf("MonthlyBudget = %d, want %d", cfg.MonthlyBudget, DefaultMonthlyBudget)
	}
	if cfg.MaxCostPerUser != DefaultMaxCostPerUser {
		t.Errorf("MaxCostPerUser = %d, want %d", cfg.MaxCostPerUser, DefaultMaxCostPerUser)
	}
}

func TestDefaultConstants(t *testing.T) {
	if DefaultDailyBudget != 100000 {
		t.Errorf("DefaultDailyBudget = %d, want 100000", DefaultDailyBudget)
	}
	if DefaultMonthlyBudget != 2000000 {
		t.Errorf("DefaultMonthlyBudget = %d, want 2000000", DefaultMonthlyBudget)
	}
	if DefaultMaxCostPerUser != 5000 {
		t.Errorf("DefaultMaxCostPerUser = %d, want 5000", DefaultMaxCostPerUser)
	}
}

func TestBudgetCheckResultDefaults(t *testing.T) {
	result := &BudgetCheckResult{}
	if result.Allowed {
		t.Error("Allowed should default to false")
	}
	if result.DailyUsed != 0 {
		t.Errorf("DailyUsed = %d, want 0", result.DailyUsed)
	}
}

func TestTokenBudgetCreation(t *testing.T) {
	cfg := DefaultBudgetConfig()
	budget := NewTokenBudget(cfg, nil, nil)
	if budget == nil {
		t.Fatal("NewTokenBudget should not return nil")
	}
	if budget.config.DailyBudget != DefaultDailyBudget {
		t.Errorf("config.DailyBudget = %d, want %d", budget.config.DailyBudget, DefaultDailyBudget)
	}
}
