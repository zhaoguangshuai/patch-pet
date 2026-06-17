// Package compliance 数据留存与合规策略
// 医疗 5 年加密 / 授权支付 7 年防篡改 / 轨迹 180 天脱敏 / 审计 1 年只读
package compliance

import (
	"fmt"
	"time"
)

// DataCategory 数据分类
type DataCategory string

const (
	CategoryMedical      DataCategory = "medical"       // 医疗数据
	CategoryPayment      DataCategory = "payment"       // 授权支付
	CategoryTrajectory   DataCategory = "trajectory"    // 宠物轨迹
	CategoryAudit        DataCategory = "audit"         // 审计日志
	CategorySession      DataCategory = "session"       // 会话数据
	CategoryUser         DataCategory = "user"          // 用户数据
)

// RetentionPolicy 数据留存策略
type RetentionPolicy struct {
	Category     DataCategory `json:"category"`
	RetentionDays int         `json:"retention_days"`
	EncryptOnStore bool       `json:"encrypt_on_store"`
	TamperProof    bool       `json:"tamper_proof"`    // 防篡改
	ReadOnly       bool       `json:"read_only"`       // 只读
	DesensitizeOnExpire bool `json:"desensitize_on_expire"` // 过期脱敏
	ArchiveDays    int        `json:"archive_days"`    // 归档天数（归档后不可二次变更）
}

// DefaultRetentionPolicies 默认数据留存策略
var DefaultRetentionPolicies = map[DataCategory]RetentionPolicy{
	CategoryMedical: {
		Category:        CategoryMedical,
		RetentionDays:   5 * 365, // 5 年
		EncryptOnStore:  true,
		TamperProof:     false,
		ReadOnly:        false,
		DesensitizeOnExpire: true,
		ArchiveDays:     5 * 365,
	},
	CategoryPayment: {
		Category:        CategoryPayment,
		RetentionDays:   7 * 365, // 7 年
		EncryptOnStore:  true,
		TamperProof:     true,
		ReadOnly:        false,
		DesensitizeOnExpire: false,
		ArchiveDays:     7 * 365,
	},
	CategoryTrajectory: {
		Category:        CategoryTrajectory,
		RetentionDays:   180, // 180 天
		EncryptOnStore:  false,
		TamperProof:     false,
		ReadOnly:        false,
		DesensitizeOnExpire: true,
		ArchiveDays:     90, // TimescaleDB 分区归档 90 天
	},
	CategoryAudit: {
		Category:        CategoryAudit,
		RetentionDays:   365, // 1 年
		EncryptOnStore:  false,
		TamperProof:     true,
		ReadOnly:        true,
		DesensitizeOnExpire: false,
		ArchiveDays:     365,
	},
	CategorySession: {
		Category:        CategorySession,
		RetentionDays:   1, // 24 小时
		EncryptOnStore:  false,
		TamperProof:     false,
		ReadOnly:        false,
		DesensitizeOnExpire: true,
		ArchiveDays:     0,
	},
}

// RetentionManager 数据留存管理器
type RetentionManager struct {
	policies map[DataCategory]RetentionPolicy
}

// NewRetentionManager 创建数据留存管理器
func NewRetentionManager() *RetentionManager {
	policies := make(map[DataCategory]RetentionPolicy)
	for k, v := range DefaultRetentionPolicies {
		policies[k] = v
	}
	return &RetentionManager{policies: policies}
}

// GetPolicy 获取数据分类的留存策略
func (m *RetentionManager) GetPolicy(category DataCategory) (RetentionPolicy, error) {
	policy, ok := m.policies[category]
	if !ok {
		return RetentionPolicy{}, fmt.Errorf("未知数据分类: %s", category)
	}
	return policy, nil
}

// IsExpired 检查数据是否过期
func (m *RetentionManager) IsExpired(category DataCategory, createdAt time.Time) (bool, error) {
	policy, err := m.GetPolicy(category)
	if err != nil {
		return false, err
	}

	expiry := createdAt.AddDate(0, 0, policy.RetentionDays)
	return time.Now().After(expiry), nil
}

// ShouldArchive 检查数据是否应归档
func (m *RetentionManager) ShouldArchive(category DataCategory, createdAt time.Time) (bool, error) {
	policy, err := m.GetPolicy(category)
	if err != nil {
		return false, err
	}

	if policy.ArchiveDays <= 0 {
		return false, nil
	}

	archiveAt := createdAt.AddDate(0, 0, policy.ArchiveDays)
	return time.Now().After(archiveAt), nil
}

// NeedsEncryption 检查数据是否需要加密存储
func (m *RetentionManager) NeedsEncryption(category DataCategory) (bool, error) {
	policy, err := m.GetPolicy(category)
	if err != nil {
		return false, err
	}
	return policy.EncryptOnStore, nil
}

// IsTamperProof 检查数据是否需要防篡改
func (m *RetentionManager) IsTamperProof(category DataCategory) (bool, error) {
	policy, err := m.GetPolicy(category)
	if err != nil {
		return false, err
	}
	return policy.TamperProof, nil
}

// UserDeletionSOP 用户注销 SOP
// 7 工作日内：脱敏 → 归档 → 删除
type UserDeletionSOP struct {
	manager *RetentionManager
}

// NewUserDeletionSOP 创建用户注销 SOP
func NewUserDeletionSOP(manager *RetentionManager) *UserDeletionSOP {
	return &UserDeletionSOP{manager: manager}
}

// DeletionStep 注销步骤
type DeletionStep struct {
	Step        int    `json:"step"`
	Description string `json:"description"`
	DaysOffset  int    `json:"days_offset"`
	Status      string `json:"status"`
}

// GetDeletionPlan 获取用户注销计划
func (s *UserDeletionSOP) GetDeletionPlan(userID string) []DeletionStep {
	return []DeletionStep{
		{Step: 1, Description: "标记用户为注销中状态", DaysOffset: 0, Status: "pending"},
		{Step: 2, Description: "脱敏用户个人信息（手机号/地址/医疗隐私）", DaysOffset: 1, Status: "pending"},
		{Step: 3, Description: "归档用户数据到冷存储", DaysOffset: 3, Status: "pending"},
		{Step: 4, Description: "删除用户主数据（保留合规要求的审计/支付记录）", DaysOffset: 5, Status: "pending"},
		{Step: 5, Description: "清理 Redis 会话/缓存", DaysOffset: 5, Status: "pending"},
		{Step: 6, Description: "确认注销完成，发送通知", DaysOffset: 7, Status: "pending"},
	}
}
