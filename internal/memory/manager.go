// Package memory MemoryManager 实现
// Session → Redis（24h TTL），Episode → PostgreSQL（180d），Family → PG + Redis 缓存
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/patch-pet/patch-pet/internal/audit"
	"github.com/patch-pet/patch-pet/internal/database"
	"github.com/patch-pet/patch-pet/internal/lifeflow"
	"github.com/patch-pet/patch-pet/pkg/logger"
	"github.com/patch-pet/patch-pet/pkg/thirdparty"
	"github.com/patch-pet/patch-pet/pkg/types"
	"github.com/patch-pet/patch-pet/pkg/utils"
)

const (
	// SessionTTL 会话记忆过期时间
	SessionTTL = 24 * time.Hour
	// EpisodeRetentionDays 流程记忆保留天数
	EpisodeRetentionDays = 180
	// FamilyCacheTTL 家庭记忆缓存过期时间
	FamilyCacheTTL = 1 * time.Hour
	// sessionKeyPrefix Redis 会话键前缀
	sessionKeyPrefix = "memory:session:"
	// familyCacheKeyPrefix Redis 家庭缓存键前缀
	familyCacheKeyPrefix = "memory:family:"
)

// EpisodeMemoryEntity 流程记忆数据库实体
type EpisodeMemoryEntity struct {
	ID          string        `json:"id" gorm:"column:id;primaryKey;type:varchar(64)"`
	AggregateID string        `json:"aggregate_id" gorm:"column:aggregate_id;type:varchar(64);not null;uniqueIndex:idx_episode_memory_aggregate_id_deleted_at"`
	DataJSON    string        `json:"data_json" gorm:"column:data_json;type:jsonb"`
	CreatedAt   types.CSTTime `json:"created_at" gorm:"column:created_at;type:timestamptz;not null"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"column:deleted_at;index:idx_episode_memory_deleted_at"`
}

func (EpisodeMemoryEntity) TableName() string { return "episode_memory" }

// FamilyMemoryEntity 家庭记忆数据库实体
type FamilyMemoryEntity struct {
	ID        string        `json:"id" gorm:"column:id;primaryKey;type:varchar(64)"`
	FamilyID  string        `json:"family_id" gorm:"column:family_id;type:varchar(64);not null;uniqueIndex:idx_family_memory_family_id_deleted_at"`
	ConfigJSON string       `json:"config_json" gorm:"column:config_json;type:jsonb"`
	CreatedAt types.CSTTime `json:"created_at" gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt types.CSTTime `json:"updated_at" gorm:"column:updated_at;type:timestamptz;not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"column:deleted_at;index:idx_family_memory_deleted_at"`
}

func (FamilyMemoryEntity) TableName() string { return "family_memory" }

// MemoryManager 三层记忆管理器实现
type MemoryManager struct {
	db       *database.DB
	redis    *thirdparty.RedisClient
	lifeflow lifeflow.Writer
	auditLog audit.Recorder
}

// NewMemoryManager 创建记忆管理器
func NewMemoryManager(db *database.DB, redis *thirdparty.RedisClient, lf lifeflow.Writer, ar audit.Recorder) *MemoryManager {
	return &MemoryManager{
		db:       db,
		redis:    redis,
		lifeflow: lf,
		auditLog: ar,
	}
}

// AutoMigrate 自动迁移数据库表
func (m *MemoryManager) AutoMigrate() error {
	return m.db.AutoMigrate(&EpisodeMemoryEntity{}, &FamilyMemoryEntity{})
}

// --- Session Memory (Redis, 24h TTL) ---

// SetSession 设置会话记忆（写入 Redis，24h 过期）
func (m *MemoryManager) SetSession(ctx context.Context, mem SessionMemory) error {
	key := sessionKeyPrefix + mem.SessionID

	data, err := json.Marshal(mem)
	if err != nil {
		return fmt.Errorf("序列化会话记忆失败: %w", err)
	}

	if err := m.redis.Set(ctx, key, string(data), SessionTTL); err != nil {
		return fmt.Errorf("写入会话记忆失败: %w", err)
	}

	logger.Debug("会话记忆已写入",
		zap.String("session_id", mem.SessionID),
		zap.Duration("ttl", SessionTTL),
	)
	return nil
}

// GetSession 获取会话记忆（从 Redis 读取）
func (m *MemoryManager) GetSession(ctx context.Context, sessionID string) (*SessionMemory, error) {
	key := sessionKeyPrefix + sessionID

	data, err := m.redis.Get(ctx, key)
	if err != nil {
		// Redis key 不存在不算错误
		return nil, nil
	}

	var mem SessionMemory
	if err := json.Unmarshal([]byte(data), &mem); err != nil {
		return nil, fmt.Errorf("反序列化会话记忆失败: %w", err)
	}

	return &mem, nil
}

// --- Episode Memory (PostgreSQL, 180d retention) ---

// SetEpisode 设置流程记忆（写入 PostgreSQL）
func (m *MemoryManager) SetEpisode(ctx context.Context, mem EpisodeMemory) error {
	dataJSON, err := json.Marshal(mem.Data)
	if err != nil {
		return fmt.Errorf("序列化流程记忆失败: %w", err)
	}

	entity := EpisodeMemoryEntity{
		ID:          utils.GenerateULID(types.IDPrefix("evt")),
		AggregateID: mem.AggregateID,
		DataJSON:    string(dataJSON),
	}

	// Upsert: 先查后更新
	existing, err := m.getEpisodeEntity(ctx, mem.AggregateID)
	if err != nil {
		return err
	}

	if existing != nil {
		existing.DataJSON = string(dataJSON)
		if err := m.db.WithContext(ctx).Save(existing).Error; err != nil {
			return fmt.Errorf("更新流程记忆失败: %w", err)
		}
	} else {
		if err := m.db.WithContext(ctx).Create(&entity).Error; err != nil {
			return fmt.Errorf("创建流程记忆失败: %w", err)
		}
	}

	// 写 lifeflow 事件
	if m.lifeflow != nil {
		_ = lifeflow.WriteAgentAction(ctx, m.lifeflow, mem.AggregateID, "memory", "set_episode",
			map[string]any{"aggregate_id": mem.AggregateID})
	}

	return nil
}

// GetEpisode 获取流程记忆（从 PostgreSQL 读取）
func (m *MemoryManager) GetEpisode(ctx context.Context, aggID string) (*EpisodeMemory, error) {
	entity, err := m.getEpisodeEntity(ctx, aggID)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, nil
	}

	var data map[string]any
	if entity.DataJSON != "" {
		if err := json.Unmarshal([]byte(entity.DataJSON), &data); err != nil {
			return nil, fmt.Errorf("反序列化流程记忆失败: %w", err)
		}
	}

	return &EpisodeMemory{
		AggregateID: entity.AggregateID,
		Data:        data,
		CreateTime:  entity.CreatedAt.Unix(),
	}, nil
}

func (m *MemoryManager) getEpisodeEntity(ctx context.Context, aggID string) (*EpisodeMemoryEntity, error) {
	var entity EpisodeMemoryEntity
	err := m.db.WithContext(ctx).
		Where("aggregate_id = ?", aggID).
		First(&entity).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("查询流程记忆失败: %w", err)
	}
	return &entity, nil
}

// --- Family Memory (PostgreSQL + Redis cache, permanent) ---

// SetFamily 设置家庭记忆（写入 PostgreSQL + 更新 Redis 缓存）
func (m *MemoryManager) SetFamily(ctx context.Context, mem FamilyMemory) error {
	configJSON, err := json.Marshal(mem.Config)
	if err != nil {
		return fmt.Errorf("序列化家庭记忆失败: %w", err)
	}

	entity := FamilyMemoryEntity{
		FamilyID:   mem.FamilyID,
		ConfigJSON: string(configJSON),
	}

	// Upsert
	existing, err := m.getFamilyEntity(ctx, mem.FamilyID)
	if err != nil {
		return err
	}

	if existing != nil {
		existing.ConfigJSON = string(configJSON)
		if err := m.db.WithContext(ctx).Save(existing).Error; err != nil {
			return fmt.Errorf("更新家庭记忆失败: %w", err)
		}
	} else {
		entity.ID = utils.GenerateULID(types.IDPrefix("evt"))
		if err := m.db.WithContext(ctx).Create(&entity).Error; err != nil {
			return fmt.Errorf("创建家庭记忆失败: %w", err)
		}
	}

	// 更新 Redis 缓存
	cacheKey := familyCacheKeyPrefix + mem.FamilyID
	cacheData, _ := json.Marshal(mem)
	_ = m.redis.Set(ctx, cacheKey, string(cacheData), FamilyCacheTTL)

	// 写审计日志（家庭记忆变更属于敏感操作）
	if m.auditLog != nil {
		_ = m.auditLog.Record(ctx, audit.Entry{
			Action:   audit.ActionToolExecute,
			Level:    audit.LevelInfo,
			Module:   "memory",
			EntityID: mem.FamilyID,
			Detail: map[string]any{
				"operation": "set_family_memory",
			},
		})
	}

	return nil
}

// GetFamily 获取家庭记忆（优先 Redis 缓存，回源 PostgreSQL）
func (m *MemoryManager) GetFamily(ctx context.Context, familyID string) (*FamilyMemory, error) {
	cacheKey := familyCacheKeyPrefix + familyID

	// 先查 Redis 缓存
	cached, err := m.redis.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		var mem FamilyMemory
		if err := json.Unmarshal([]byte(cached), &mem); err == nil {
			return &mem, nil
		}
	}

	// 缓存未命中，查 PostgreSQL
	entity, err := m.getFamilyEntity(ctx, familyID)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, nil
	}

	var config map[string]any
	if entity.ConfigJSON != "" {
		if err := json.Unmarshal([]byte(entity.ConfigJSON), &config); err != nil {
			return nil, fmt.Errorf("反序列化家庭记忆失败: %w", err)
		}
	}

	mem := &FamilyMemory{
		FamilyID: entity.FamilyID,
		Config:   config,
	}

	// 回填 Redis 缓存
	cacheData, _ := json.Marshal(mem)
	_ = m.redis.Set(ctx, cacheKey, string(cacheData), FamilyCacheTTL)

	return mem, nil
}

func (m *MemoryManager) getFamilyEntity(ctx context.Context, familyID string) (*FamilyMemoryEntity, error) {
	var entity FamilyMemoryEntity
	err := m.db.WithContext(ctx).
		Where("family_id = ?", familyID).
		First(&entity).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("查询家庭记忆失败: %w", err)
	}
	return &entity, nil
}
