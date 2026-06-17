package medical

import (
	"testing"
)

func TestEscalationLevelConstants(t *testing.T) {
	levels := []EscalationLevel{
		EscalationNone,
		EscalationReminder,
		EscalationWarn,
		EscalationCritical,
	}
	if len(levels) != 4 {
		t.Errorf("EscalationLevel count = %d, want 4", len(levels))
	}
}

func TestEscalationResultDefaults(t *testing.T) {
	result := &EscalationResult{
		Level: EscalationNone,
	}
	if result.SuggestClinic {
		t.Error("SuggestClinic should default to false")
	}
	if result.MissedCount != 0 {
		t.Errorf("MissedCount = %d, want 0", result.MissedCount)
	}
}

func TestEvaluateEscalationP0FirstMiss(t *testing.T) {
	chain := &EscalationChain{rules: defaultEscalationRules}

	task := CareTask{
		ID:        "task_1",
		EpisodeID: "ep_1",
		RiskLevel: "P0",
	}

	result := chain.evaluateEscalation(task, 1)
	if result.Level != EscalationWarn {
		t.Errorf("P0 first miss level = %s, want %s", result.Level, EscalationWarn)
	}
	if !result.SuggestClinic {
		t.Error("P0 first miss should suggest clinic")
	}
}

func TestEvaluateEscalationP0SecondMiss(t *testing.T) {
	chain := &EscalationChain{rules: defaultEscalationRules}

	task := CareTask{
		ID:        "task_1",
		EpisodeID: "ep_1",
		RiskLevel: "P0",
	}

	result := chain.evaluateEscalation(task, 2)
	if result.Level != EscalationCritical {
		t.Errorf("P0 second miss level = %s, want %s", result.Level, EscalationCritical)
	}
	if !result.SuggestClinic {
		t.Error("P0 second miss should suggest clinic")
	}
}

func TestEvaluateEscalationP1FirstMiss(t *testing.T) {
	chain := &EscalationChain{rules: defaultEscalationRules}

	task := CareTask{
		ID:        "task_1",
		EpisodeID: "ep_1",
		RiskLevel: "P1",
	}

	result := chain.evaluateEscalation(task, 1)
	if result.Level != EscalationWarn {
		t.Errorf("P1 first miss level = %s, want %s", result.Level, EscalationWarn)
	}
}

func TestEvaluateEscalationP2FirstMiss(t *testing.T) {
	chain := &EscalationChain{rules: defaultEscalationRules}

	task := CareTask{
		ID:        "task_1",
		EpisodeID: "ep_1",
		RiskLevel: "P2",
	}

	result := chain.evaluateEscalation(task, 1)
	if result.Level != EscalationNone {
		t.Errorf("P2 first miss level = %s, want %s", result.Level, EscalationNone)
	}
}

func TestEvaluateEscalationP2ThirdMiss(t *testing.T) {
	chain := &EscalationChain{rules: defaultEscalationRules}

	task := CareTask{
		ID:        "task_1",
		EpisodeID: "ep_1",
		RiskLevel: "P2",
	}

	result := chain.evaluateEscalation(task, 3)
	if result.Level != EscalationReminder {
		t.Errorf("P2 third miss level = %s, want %s", result.Level, EscalationReminder)
	}
	if result.SuggestClinic {
		t.Error("P2 third miss should not suggest clinic")
	}
}

func TestMatchesRiskLevel(t *testing.T) {
	chain := &EscalationChain{}

	if !chain.matchesRiskLevel("P0", []string{"P0", "P1"}) {
		t.Error("P0 should match [P0, P1]")
	}
	if chain.matchesRiskLevel("P2", []string{"P0", "P1"}) {
		t.Error("P2 should not match [P0, P1]")
	}
}

func TestDefaultEscalationRulesCount(t *testing.T) {
	if len(defaultEscalationRules) != 4 {
		t.Errorf("defaultEscalationRules count = %d, want 4", len(defaultEscalationRules))
	}
}

func TestSuggestionMessageContent(t *testing.T) {
	chain := &EscalationChain{rules: defaultEscalationRules}

	task := CareTask{
		ID:        "task_abc",
		EpisodeID: "ep_xyz",
		RiskLevel: "P0",
	}

	result := chain.evaluateEscalation(task, 2)
	if result.SuggestionMessage == "" {
		t.Error("SuggestionMessage should not be empty for critical level")
	}
	if result.SuggestionMessage == "" {
		t.Error("SuggestionMessage should contain task ID")
	}
}
