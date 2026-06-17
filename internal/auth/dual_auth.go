package auth

import (
	"context"
	"fmt"

	"github.com/patch-pet/patch-pet/internal/policy"
)

// AuthLayer 鉴权层级
type AuthLayer string

const (
	LayerAgent AuthLayer = "agent" // Agent 调用能力校验
	LayerUser  AuthLayer = "user"  // 用户操作权限校验
)

// DualAuthRequest 双层鉴权请求
type DualAuthRequest struct {
	// Agent 层
	ToolName   string // 工具名称
	Permission string // 工具所需权限

	// User 层
	UserCtx    *RequestContext // 用户上下文
	Resource   string          // 资源标识（如 medical_episode:ep_001）
	Action     string          // 操作类型（read / write / delete / share）
}

// DualAuthResult 双层鉴权结果
type DualAuthResult struct {
	Allowed    bool       `json:"allowed"`
	DeniedAt   AuthLayer  `json:"denied_at"`   // 在哪一层被拒绝
	Reason     string     `json:"reason"`       // 拒绝原因
	AgentAllow bool       `json:"agent_allow"`  // Agent 层是否通过
	UserAllow  bool       `json:"user_allow"`   // User 层是否通过
}

// DualAuthenticator 双层鉴权器
// Agent 调用能力 ≠ 用户操作权限，独立校验
// 两层都通过才放行
type DualAuthenticator struct {
	policyEngine *policy.Engine
	userAuth     Authenticator
}

// NewDualAuthenticator 创建双层鉴权器
func NewDualAuthenticator(pe *policy.Engine, ua Authenticator) *DualAuthenticator {
	return &DualAuthenticator{
		policyEngine: pe,
		userAuth:     ua,
	}
}

// Authenticate 双层鉴权
// 1. Agent 层：校验工具调用权限（policy engine）
// 2. User 层：校验用户数据操作权限（authenticator）
// 两层都通过才返回 allowed
func (d *DualAuthenticator) Authenticate(ctx context.Context, req DualAuthRequest) (*DualAuthResult, error) {
	result := &DualAuthResult{}

	// Layer 1: Agent 调用能力校验
	agentDecision := d.policyEngine.Evaluate(map[string]string{
		"permission": req.Permission,
		"tool":       req.ToolName,
	})
	result.AgentAllow = agentDecision.Allow

	if !agentDecision.Allow {
		result.Allowed = false
		result.DeniedAt = LayerAgent
		result.Reason = fmt.Sprintf("Agent 调用被拒: %s", agentDecision.Msg)
		return result, nil
	}

	// Layer 2: 用户操作权限校验
	if req.UserCtx != nil && d.userAuth != nil {
		if err := d.userAuth.Authorize(ctx, req.UserCtx, req.Resource, req.Action); err != nil {
			result.UserAllow = false
			result.Allowed = false
			result.DeniedAt = LayerUser
			result.Reason = fmt.Sprintf("用户权限被拒: %v", err)
			return result, nil
		}
		result.UserAllow = true
	} else {
		// 无用户上下文时（系统级调用），跳过用户层校验
		result.UserAllow = true
	}

	result.Allowed = true
	return result, nil
}

// AuthorizeUser 仅校验用户权限（用于非 Agent 调用场景）
func (d *DualAuthenticator) AuthorizeUser(ctx context.Context, reqCtx *RequestContext, resource, action string) error {
	if d.userAuth == nil {
		return fmt.Errorf("认证器未初始化")
	}
	return d.userAuth.Authorize(ctx, reqCtx, resource, action)
}

// EvaluateAgent 仅校验 Agent 能力（用于工具网关前置校验）
func (d *DualAuthenticator) EvaluateAgent(toolName, permission string) policy.Decision {
	return d.policyEngine.Evaluate(map[string]string{
		"permission": permission,
		"tool":       toolName,
	})
}
