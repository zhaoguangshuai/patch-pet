package agent

import (
	"context"
	"testing"

	"github.com/patch-pet/patch-pet/internal/audit"
	"github.com/patch-pet/patch-pet/pkg/types"
)

// mockAgent 测试用 Agent
type mockAgent struct {
	agentType AgentType
}

func (m *mockAgent) Type() AgentType { return m.agentType }

func (m *mockAgent) Handle(ctx context.Context, input string) (*types.ToolIntent, error) {
	return &types.ToolIntent{
		Intent: "mock",
		Action: types.LLMActionNotice,
	}, nil
}

// noopAudit 空审计记录器
type noopAudit struct{}

func (n *noopAudit) Record(ctx context.Context, entry audit.Entry) error { return nil }

func TestClassifyMedicalIntent(t *testing.T) {
	r := NewIntentRouter(nil)

	tests := []string{
		"我的狗需要吃药",
		"宠物的症状是呕吐",
		"帮我查看 medical_episode",
		"疫苗接种时间",
	}
	for _, input := range tests {
		cats := r.classifyIntent(input)
		if len(cats) == 0 || cats[0] != IntentMedical {
			t.Errorf("classifyIntent(%q) = %v, want [medical]", input, cats)
		}
	}
}

func TestClassifyDogwalkIntent(t *testing.T) {
	r := NewIntentRouter(nil)

	tests := []string{
		"帮我遛狗",
		"预约代遛服务",
		"查看 dog_walk_order",
		"规划散步路线",
	}
	for _, input := range tests {
		cats := r.classifyIntent(input)
		if len(cats) == 0 || cats[0] != IntentDogwalk {
			t.Errorf("classifyIntent(%q) = %v, want [dogwalk]", input, cats)
		}
	}
}

func TestClassifyGeneralIntent(t *testing.T) {
	r := NewIntentRouter(nil)

	cats := r.classifyIntent("今天天气怎么样")
	if len(cats) != 0 {
		t.Errorf("classifyIntent(天气) = %v, want empty", cats)
	}
}

func TestClassifyConflictIntent(t *testing.T) {
	r := NewIntentRouter(nil)

	// 同时匹配医疗和代遛
	cats := r.classifyIntent("遛狗回来后发现宠物呕吐需要看症状")
	hasMedical := false
	hasDogwalk := false
	for _, c := range cats {
		if c == IntentMedical {
			hasMedical = true
		}
		if c == IntentDogwalk {
			hasDogwalk = true
		}
	}
	if !hasMedical {
		t.Error("should detect medical intent")
	}
	if !hasDogwalk {
		t.Error("should detect dogwalk intent")
	}
}

func TestRouteMedicalPriority(t *testing.T) {
	r := NewIntentRouter(nil)
	medicalAgent := &mockAgent{agentType: AgentTypeMedical}
	dogwalkAgent := &mockAgent{agentType: AgentTypeDogwalk}
	r.RegisterAgent(IntentMedical, medicalAgent)
	r.RegisterAgent(IntentDogwalk, dogwalkAgent)

	// 冲突场景：医疗优先
	result, err := r.Route(context.Background(), "遛狗后宠物呕吐需要诊断")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Category != IntentMedical {
		t.Errorf("category = %s, want medical", result.Category)
	}
	if !result.Conflict {
		t.Error("should detect conflict")
	}
}

func TestRouteNoConflict(t *testing.T) {
	r := NewIntentRouter(nil)
	r.RegisterAgent(IntentMedical, &mockAgent{agentType: AgentTypeMedical})
	r.RegisterAgent(IntentDogwalk, &mockAgent{agentType: AgentTypeDogwalk})

	result, err := r.Route(context.Background(), "帮我预约遛狗服务")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Category != IntentDogwalk {
		t.Errorf("category = %s, want dogwalk", result.Category)
	}
	if result.Conflict {
		t.Error("should not detect conflict")
	}
}

func TestRouteGeneralFallback(t *testing.T) {
	r := NewIntentRouter(nil)
	r.RegisterAgent(IntentMedical, &mockAgent{agentType: AgentTypeMedical})

	result, err := r.Route(context.Background(), "你好")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Category != IntentMedical {
		t.Errorf("category = %s, want medical (fallback)", result.Category)
	}
}

func TestRouteNoAgentAvailable(t *testing.T) {
	r := NewIntentRouter(nil)

	_, err := r.Route(context.Background(), "你好")
	if err == nil {
		t.Error("expected error when no agents registered")
	}
}

func TestSortByPriority(t *testing.T) {
	r := NewIntentRouter(nil)
	cats := []IntentCategory{IntentGeneral, IntentDogwalk, IntentMedical}
	r.sortByPriority(cats)

	if cats[0] != IntentMedical {
		t.Errorf("cats[0] = %s, want medical", cats[0])
	}
	if cats[1] != IntentDogwalk {
		t.Errorf("cats[1] = %s, want dogwalk", cats[1])
	}
	if cats[2] != IntentGeneral {
		t.Errorf("cats[2] = %s, want general", cats[2])
	}
}
