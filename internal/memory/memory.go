// Package memory Memory 三层记忆管理
// Session（Redis 24h）/ Episode（PG 流程结束+180d）/ Family（PG+Redis 永久）
// 物理隔离，跨层禁止直接复制
package memory

import "context"

// SessionMemory 会话记忆（Redis，24h 超时自动销毁）
type SessionMemory struct {
	SessionID string         `json:"session_id"`
	Context   map[string]any `json:"context"`
	ExpireAt  int64          `json:"expire_at"` // Unix 时间戳
}

// EpisodeMemory 流程记忆（PostgreSQL，流程结束 + 180 天）
type EpisodeMemory struct {
	AggregateID string         `json:"aggregate_id"`
	Data        map[string]any `json:"data"`
	CreateTime  int64          `json:"create_time"`
}

// FamilyMemory 家庭长期记忆（PostgreSQL + Redis 缓存，永久）
type FamilyMemory struct {
	FamilyID string         `json:"family_id"`
	Config   map[string]any `json:"config"`
}

// Manager Memory 管理器接口
// 读取前必须经过权限校验，变更必须同步写入生命流 + 审计日志
type Manager interface {
	SetSession(ctx context.Context, mem SessionMemory) error
	GetSession(ctx context.Context, sessionID string) (*SessionMemory, error)
	SetEpisode(ctx context.Context, mem EpisodeMemory) error
	GetEpisode(ctx context.Context, aggID string) (*EpisodeMemory, error)
	SetFamily(ctx context.Context, mem FamilyMemory) error
	GetFamily(ctx context.Context, familyID string) (*FamilyMemory, error)
}
