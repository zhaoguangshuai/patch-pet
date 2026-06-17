package auth

import (
	"context"
	"fmt"
	"testing"

	"github.com/patch-pet/patch-pet/internal/policy"
)

// mockAuthenticator 测试用认证器
type mockAuthenticator struct {
	allowResources map[string]bool
}

func (m *mockAuthenticator) Authenticate(ctx context.Context, token string) (*RequestContext, error) {
	return &RequestContext{
		UserID:   "user_001",
		FamilyID: "fam_001",
		Role:     RoleOwner,
	}, nil
}

func (m *mockAuthenticator) Authorize(ctx context.Context, reqCtx *RequestContext, resource string, action string) error {
	key := resource + ":" + action
	if m.allowResources[key] {
		return nil
	}
	return fmt.Errorf("用户无权限: %s %s", resource, action)
}

func setupTestDualAuth() *DualAuthenticator {
	pe := policy.New()
	pe.AddRule(policy.Rule{
		ID:       "allow-medical-read",
		Name:     "允许读取医疗数据",
		When:     map[string]string{"permission": "medical.read"},
		Action:   "allow",
		Priority: 10,
	})
	pe.AddRule(policy.Rule{
		ID:       "allow-dogwalk-read",
		Name:     "允许读取代遛数据",
		When:     map[string]string{"permission": "dogwalk.read"},
		Action:   "allow",
		Priority: 10,
	})

	ua := &mockAuthenticator{
		allowResources: map[string]bool{
			"medical_episode:ep_001:read": true,
			"dog_walk_order:order_001:read": true,
		},
	}

	return NewDualAuthenticator(pe, ua)
}

func TestDualAuthBothAllowed(t *testing.T) {
	da := setupTestDualAuth()

	result, err := da.Authenticate(context.Background(), DualAuthRequest{
		ToolName:   "get_medical_episode",
		Permission: "medical.read",
		UserCtx: &RequestContext{
			UserID:   "user_001",
			FamilyID: "fam_001",
			Role:     RoleOwner,
		},
		Resource: "medical_episode:ep_001",
		Action:   "read",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed, got denied")
	}
	if !result.AgentAllow {
		t.Error("Agent layer should allow")
	}
	if !result.UserAllow {
		t.Error("User layer should allow")
	}
	if result.DeniedAt != "" {
		t.Errorf("DeniedAt should be empty, got %q", result.DeniedAt)
	}
}

func TestDualAuthAgentDenied(t *testing.T) {
	da := setupTestDualAuth()

	result, err := da.Authenticate(context.Background(), DualAuthRequest{
		ToolName:   "delete_medical_data",
		Permission: "medical.delete", // 无匹配规则 → Default-Deny
		UserCtx: &RequestContext{
			UserID:   "user_001",
			FamilyID: "fam_001",
			Role:     RoleOwner,
		},
		Resource: "medical_episode:ep_001",
		Action:   "delete",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("expected denied, got allowed")
	}
	if result.DeniedAt != LayerAgent {
		t.Errorf("DeniedAt = %q, want %q", result.DeniedAt, LayerAgent)
	}
	if result.AgentAllow {
		t.Error("Agent layer should deny")
	}
}

func TestDualAuthUserDenied(t *testing.T) {
	da := setupTestDualAuth()

	result, err := da.Authenticate(context.Background(), DualAuthRequest{
		ToolName:   "get_medical_episode",
		Permission: "medical.read",
		UserCtx: &RequestContext{
			UserID:   "user_002", // 不同用户
			FamilyID: "fam_001",
			Role:     RoleGuest,
		},
		Resource: "medical_episode:ep_999", // 未授权资源
		Action:   "read",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("expected denied, got allowed")
	}
	if result.DeniedAt != LayerUser {
		t.Errorf("DeniedAt = %q, want %q", result.DeniedAt, LayerUser)
	}
	if !result.AgentAllow {
		t.Error("Agent layer should allow")
	}
	if result.UserAllow {
		t.Error("User layer should deny")
	}
}

func TestDualAuthNoUserCtxSkipsUserLayer(t *testing.T) {
	da := setupTestDualAuth()

	result, err := da.Authenticate(context.Background(), DualAuthRequest{
		ToolName:   "get_medical_episode",
		Permission: "medical.read",
		UserCtx:    nil, // 系统级调用
		Resource:   "medical_episode:ep_001",
		Action:     "read",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("system-level call should be allowed when agent layer passes")
	}
	if !result.UserAllow {
		t.Error("User layer should be skipped (allowed) for system calls")
	}
}

func TestDualAuthAgentDeniedSkipsUserLayer(t *testing.T) {
	da := setupTestDualAuth()

	result, err := da.Authenticate(context.Background(), DualAuthRequest{
		ToolName:   "unknown_tool",
		Permission: "unknown.permission",
		UserCtx: &RequestContext{
			UserID: "user_001",
		},
		Resource: "some:resource",
		Action:   "read",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("should be denied at agent layer")
	}
	if result.DeniedAt != LayerAgent {
		t.Errorf("DeniedAt = %q, want %q", result.DeniedAt, LayerAgent)
	}
	// User 层不应被评估
	if result.UserAllow {
		t.Error("User layer should not be evaluated when agent denies")
	}
}

func TestEvaluateAgentOnly(t *testing.T) {
	da := setupTestDualAuth()

	dec := da.EvaluateAgent("get_medical_episode", "medical.read")
	if !dec.Allow {
		t.Error("medical.read should be allowed")
	}

	dec = da.EvaluateAgent("delete_all", "medical.delete")
	if dec.Allow {
		t.Error("medical.delete should be denied (Default-Deny)")
	}
}

func TestAuthorizeUserOnly(t *testing.T) {
	da := setupTestDualAuth()

	err := da.AuthorizeUser(context.Background(),
		&RequestContext{UserID: "user_001", FamilyID: "fam_001", Role: RoleOwner},
		"medical_episode:ep_001", "read")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	err = da.AuthorizeUser(context.Background(),
		&RequestContext{UserID: "user_002", FamilyID: "fam_001", Role: RoleGuest},
		"medical_episode:ep_999", "read")
	if err == nil {
		t.Error("expected error for unauthorized user")
	}
}
