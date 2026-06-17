package auth

import (
	"context"
	"testing"
)

func TestAuthenticateEmptyToken(t *testing.T) {
	a := NewJWTAuthenticator()
	_, err := a.Authenticate(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestAuthenticateBearerOnly(t *testing.T) {
	a := NewJWTAuthenticator()
	_, err := a.Authenticate(context.Background(), "Bearer ")
	if err == nil {
		t.Error("expected error for bearer-only token")
	}
}

func TestAuthenticateInvalidFormat(t *testing.T) {
	a := NewJWTAuthenticator()
	_, err := a.Authenticate(context.Background(), "Bearer user001|fam001")
	if err == nil {
		t.Error("expected error for token with < 3 parts")
	}
}

func TestAuthenticateInvalidRole(t *testing.T) {
	a := NewJWTAuthenticator()
	_, err := a.Authenticate(context.Background(), "Bearer user001|fam001|admin")
	if err == nil {
		t.Error("expected error for invalid role")
	}
}

func TestAuthenticateValidOwner(t *testing.T) {
	a := NewJWTAuthenticator()
	ctx, err := a.Authenticate(context.Background(), "Bearer user001|fam001|owner|app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.UserID != "user001" {
		t.Errorf("UserID = %q, want user001", ctx.UserID)
	}
	if ctx.FamilyID != "fam001" {
		t.Errorf("FamilyID = %q, want fam001", ctx.FamilyID)
	}
	if ctx.Role != RoleOwner {
		t.Errorf("Role = %q, want %q", ctx.Role, RoleOwner)
	}
	if ctx.Source != "app" {
		t.Errorf("Source = %q, want app", ctx.Source)
	}
}

func TestAuthenticateDefaultSource(t *testing.T) {
	a := NewJWTAuthenticator()
	ctx, err := a.Authenticate(context.Background(), "Bearer user001|fam001|member")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.Source != "app" {
		t.Errorf("Source = %q, want app (default)", ctx.Source)
	}
}

func TestAuthenticateGuest(t *testing.T) {
	a := NewJWTAuthenticator()
	ctx, err := a.Authenticate(context.Background(), "Bearer user001|fam001|guest|web")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.Role != RoleGuest {
		t.Errorf("Role = %q, want %q", ctx.Role, RoleGuest)
	}
	if ctx.Source != "web" {
		t.Errorf("Source = %q, want web", ctx.Source)
	}
}

func TestAuthorizeNilContext(t *testing.T) {
	a := NewJWTAuthenticator()
	err := a.Authorize(context.Background(), nil, "resource", "read")
	if err == nil {
		t.Error("expected error for nil context")
	}
}

func TestAuthorizeOwnerFullAccess(t *testing.T) {
	a := NewJWTAuthenticator()
	ctx := &RequestContext{UserID: "u1", FamilyID: "f1", Role: RoleOwner}

	actions := []string{"read", "write", "delete", "share"}
	for _, action := range actions {
		err := a.Authorize(context.Background(), ctx, "medical_episode:ep_001", action)
		if err != nil {
			t.Errorf("Owner should have %s access, got: %v", action, err)
		}
	}
}

func TestAuthorizeMemberReadAllowed(t *testing.T) {
	a := NewJWTAuthenticator()
	ctx := &RequestContext{UserID: "u1", FamilyID: "f1", Role: RoleMember}

	err := a.Authorize(context.Background(), ctx, "medical_episode:ep_001", "read")
	if err != nil {
		t.Errorf("Member should have read access, got: %v", err)
	}
}

func TestAuthorizeMemberWriteAllowed(t *testing.T) {
	a := NewJWTAuthenticator()
	ctx := &RequestContext{UserID: "u1", FamilyID: "f1", Role: RoleMember}

	err := a.Authorize(context.Background(), ctx, "care_task:task_001", "write")
	if err != nil {
		t.Errorf("Member should have write access, got: %v", err)
	}
}

func TestAuthorizeMemberDeleteDenied(t *testing.T) {
	a := NewJWTAuthenticator()
	ctx := &RequestContext{UserID: "u1", FamilyID: "f1", Role: RoleMember}

	err := a.Authorize(context.Background(), ctx, "medical_episode:ep_001", "delete")
	if err == nil {
		t.Error("Member should NOT have delete access")
	}
}

func TestAuthorizeGuestReadAllowed(t *testing.T) {
	a := NewJWTAuthenticator()
	ctx := &RequestContext{UserID: "u1", FamilyID: "f1", Role: RoleGuest}

	err := a.Authorize(context.Background(), ctx, "medical_episode:ep_001", "read")
	if err != nil {
		t.Errorf("Guest should have read access, got: %v", err)
	}
}

func TestAuthorizeGuestWriteDenied(t *testing.T) {
	a := NewJWTAuthenticator()
	ctx := &RequestContext{UserID: "u1", FamilyID: "f1", Role: RoleGuest}

	err := a.Authorize(context.Background(), ctx, "medical_episode:ep_001", "write")
	if err == nil {
		t.Error("Guest should NOT have write access")
	}
}

func TestAuthorizeGuestDeleteDenied(t *testing.T) {
	a := NewJWTAuthenticator()
	ctx := &RequestContext{UserID: "u1", FamilyID: "f1", Role: RoleGuest}

	err := a.Authorize(context.Background(), ctx, "medical_episode:ep_001", "delete")
	if err == nil {
		t.Error("Guest should NOT have delete access")
	}
}

func TestParseResource(t *testing.T) {
	tests := []struct {
		input        string
		wantType     string
		wantID       string
	}{
		{"medical_episode:ep_001", "medical_episode", "ep_001"},
		{"care_task", "care_task", ""},
		{"dog_walk_order:order_xyz", "dog_walk_order", "order_xyz"},
	}

	for _, tt := range tests {
		gotType, gotID := parseResource(tt.input)
		if gotType != tt.wantType || gotID != tt.wantID {
			t.Errorf("parseResource(%q) = (%q, %q), want (%q, %q)",
				tt.input, gotType, gotID, tt.wantType, tt.wantID)
		}
	}
}

func TestIsValidRole(t *testing.T) {
	valid := []Role{RoleOwner, RoleMember, RoleGuest}
	for _, r := range valid {
		if !isValidRole(r) {
			t.Errorf("role %q should be valid", r)
		}
	}
	if isValidRole("admin") {
		t.Error("admin should be invalid")
	}
	if isValidRole("") {
		t.Error("empty role should be invalid")
	}
}
