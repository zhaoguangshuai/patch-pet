// Package auth JWT 认证器实现
// 账号鉴权 + 家庭/成员角色 + 数据权限
package auth

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// JWTAuthenticator JWT 认证器
// Token 格式：Bearer <token>
// Token 内容：base64(user_id|family_id|role|source)
type JWTAuthenticator struct {
	secret string
}

// NewJWTAuthenticator 创建 JWT 认证器
func NewJWTAuthenticator() *JWTAuthenticator {
	return &JWTAuthenticator{
		secret: os.Getenv("JWT_SECRET"),
	}
}

// Authenticate 从 Token 解析用户身份
func (a *JWTAuthenticator) Authenticate(ctx context.Context, token string) (*RequestContext, error) {
	if token == "" {
		return nil, fmt.Errorf("Token 为空")
	}

	// 去掉 Bearer 前缀
	token = strings.TrimPrefix(token, "Bearer ")
	token = strings.TrimSpace(token)

	if token == "" {
		return nil, fmt.Errorf("Token 格式无效")
	}

	// 简化解析：实际生产应使用 JWT 库验证签名
	// 格式：user_id|family_id|role|source
	parts := strings.Split(token, "|")
	if len(parts) < 3 {
		return nil, fmt.Errorf("Token 格式无效: 需要 user_id|family_id|role 格式")
	}

	role := Role(parts[2])
	if !isValidRole(role) {
		return nil, fmt.Errorf("无效的角色: %s", role)
	}

	source := "app"
	if len(parts) >= 4 {
		source = parts[3]
	}

	return &RequestContext{
		UserID:   parts[0],
		FamilyID: parts[1],
		Role:     role,
		Source:   source,
	}, nil
}

// Authorize 校验用户是否有指定资源的访问权限
func (a *JWTAuthenticator) Authorize(ctx context.Context, reqCtx *RequestContext, resource string, action string) error {
	if reqCtx == nil {
		return fmt.Errorf("请求上下文为空")
	}

	// 解析资源类型和 ID
	resourceType, resourceID := parseResource(resource)

	// 基于角色的权限校验
	switch reqCtx.Role {
	case RoleOwner:
		// 家庭主人：所有权限
		return nil

	case RoleMember:
		// 家庭成员：读取全部，写入部分受限
		if action == "delete" {
			return fmt.Errorf("成员角色无删除权限: resource=%s", resource)
		}
		// 成员只能操作自己家庭的数据
		if !a.checkFamilyAccess(reqCtx, resourceType, resourceID) {
			return fmt.Errorf("无权访问其他家庭数据: resource=%s", resource)
		}
		return nil

	case RoleGuest:
		// 访客：仅读取权限
		if action != "read" {
			return fmt.Errorf("访客仅有读取权限: action=%s", action)
		}
		return nil

	default:
		return fmt.Errorf("未知角色: %s", reqCtx.Role)
	}
}

// checkFamilyAccess 检查家庭数据访问权限
func (a *JWTAuthenticator) checkFamilyAccess(reqCtx *RequestContext, resourceType, resourceID string) bool {
	// 实际实现应查询数据库验证资源所属家庭
	// 当前简化为放行（需结合实际数据模型）
	return true
}

// parseResource 解析资源标识
// 格式：resource_type:resource_id 或 resource_type
func parseResource(resource string) (string, string) {
	parts := strings.SplitN(resource, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}

// isValidRole 校验角色是否有效
func isValidRole(role Role) bool {
	switch role {
	case RoleOwner, RoleMember, RoleGuest:
		return true
	default:
		return false
	}
}
