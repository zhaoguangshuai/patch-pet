// Package auth 认证授权
// 双层鉴权：Agent 调用能力 ≠ 用户操作权限，独立校验
package auth

import "context"

// Role 家庭成员角色
type Role string

const (
	RoleOwner  Role = "owner"  // 家庭主人
	RoleMember Role = "member" // 家庭成员
	RoleGuest  Role = "guest"  // 访客
)

// RequestContext 请求上下文（从 Authorization Header 解析）
type RequestContext struct {
	UserID    string `json:"user_id"`
	FamilyID  string `json:"family_id"`
	Role      Role   `json:"role"`
	TraceID   string `json:"trace_id"`
	Source    string `json:"source"` // app / web / clinic / vendor
}

// Authenticator 认证器接口
type Authenticator interface {
	// Authenticate 从 Token 解析用户身份
	Authenticate(ctx context.Context, token string) (*RequestContext, error)
	// Authorize 校验用户是否有指定资源的访问权限
	Authorize(ctx context.Context, reqCtx *RequestContext, resource string, action string) error
}
