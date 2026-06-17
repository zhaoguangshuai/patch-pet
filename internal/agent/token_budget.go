// Package agent Token 预算管理
// DAILY=100000 / MONTHLY=2000000 / MAX_COST_PER_USER=5000
// 超阈 P2 告警，达上限切静态模板
package agent

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/patch-pet/patch-pet/internal/audit"
	"github.com/patch-pet/patch-pet/pkg/logger"
	"github.com/patch-pet/patch-pet/pkg/thirdparty"
)

const (
	// DefaultDailyBudget 每日 Token 预算
	DefaultDailyBudget = 100000
	// DefaultMonthlyBudget 每月 Token 预算
	DefaultMonthlyBudget = 2000000
	// DefaultMaxCostPerUser 单用户最大 Token 消耗
	DefaultMaxCostPerUser = 5000

	// tokenBudgetKeyPrefix Redis 键前缀
	tokenBudgetKeyPrefix = "token:budget:"
	// tokenUserKeyPrefix 用户 Token 消耗键前缀
	tokenUserKeyPrefix = "token:user:"
)

// BudgetConfig Token 预算配置
type BudgetConfig struct {
	DailyBudget     int // 每日 Token 预算
	MonthlyBudget   int // 每月 Token 预算
	MaxCostPerUser  int // 单用户最大 Token 消耗
}

// DefaultBudgetConfig 默认预算配置
func DefaultBudgetConfig() BudgetConfig {
	return BudgetConfig{
		DailyBudget:    DefaultDailyBudget,
		MonthlyBudget:  DefaultMonthlyBudget,
		MaxCostPerUser: DefaultMaxCostPerUser,
	}
}

// BudgetCheckResult 预算检查结果
type BudgetCheckResult struct {
	Allowed     bool   `json:"allowed"`      // 是否允许调用
	Reason      string `json:"reason"`       // 拒绝原因
	DailyUsed   int    `json:"daily_used"`   // 今日已用
	MonthlyUsed int    `json:"monthly_used"` // 本月已用
	UserUsed    int    `json:"user_used"`    // 用户已用
}

// TokenBudget Token 预算管理器
type TokenBudget struct {
	config   BudgetConfig
	redis    *thirdparty.RedisClient
	auditLog audit.Recorder
}

// NewTokenBudget 创建 Token 预算管理器
func NewTokenBudget(cfg BudgetConfig, redis *thirdparty.RedisClient, ar audit.Recorder) *TokenBudget {
	return &TokenBudget{
		config:   cfg,
		redis:    redis,
		auditLog: ar,
	}
}

// CheckBudget 检查预算是否允许调用
func (b *TokenBudget) CheckBudget(ctx context.Context, userID string) (*BudgetCheckResult, error) {
	// 获取今日已用
	dailyUsed, err := b.getDailyUsed(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取每日用量失败: %w", err)
	}

	// 获取本月已用
	monthlyUsed, err := b.getMonthlyUsed(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取每月用量失败: %w", err)
	}

	// 获取用户已用
	userUsed, err := b.getUserUsed(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户用量失败: %w", err)
	}

	result := &BudgetCheckResult{
		DailyUsed:   dailyUsed,
		MonthlyUsed: monthlyUsed,
		UserUsed:    userUsed,
	}

	// 检查每日预算
	if dailyUsed >= b.config.DailyBudget {
		result.Allowed = false
		result.Reason = fmt.Sprintf("每日 Token 预算已耗尽 (%d/%d)", dailyUsed, b.config.DailyBudget)
		b.recordBudgetExceeded(ctx, "daily", userID, dailyUsed, b.config.DailyBudget)
		return result, nil
	}

	// 检查每月预算
	if monthlyUsed >= b.config.MonthlyBudget {
		result.Allowed = false
		result.Reason = fmt.Sprintf("每月 Token 预算已耗尽 (%d/%d)", monthlyUsed, b.config.MonthlyBudget)
		b.recordBudgetExceeded(ctx, "monthly", userID, monthlyUsed, b.config.MonthlyBudget)
		return result, nil
	}

	// 检查用户预算
	if userUsed >= b.config.MaxCostPerUser {
		result.Allowed = false
		result.Reason = fmt.Sprintf("用户 Token 消耗已达上限 (%d/%d)", userUsed, b.config.MaxCostPerUser)
		b.recordBudgetExceeded(ctx, "user", userID, userUsed, b.config.MaxCostPerUser)
		return result, nil
	}

	// 检查是否接近阈值（80%）→ P2 告警
	b.checkThreshold(ctx, "daily", dailyUsed, b.config.DailyBudget, userID)
	b.checkThreshold(ctx, "monthly", monthlyUsed, b.config.MonthlyBudget, userID)

	result.Allowed = true
	return result, nil
}

// RecordUsage 记录 Token 消耗
func (b *TokenBudget) RecordUsage(ctx context.Context, userID string, tokens int) error {
	if tokens <= 0 {
		return nil
	}

	// 增加每日用量
	if err := b.incrementDaily(ctx, tokens); err != nil {
		return err
	}

	// 增加每月用量
	if err := b.incrementMonthly(ctx, tokens); err != nil {
		return err
	}

	// 增加用户用量
	if err := b.incrementUser(ctx, userID, tokens); err != nil {
		return err
	}

	return nil
}

func (b *TokenBudget) getDailyUsed(ctx context.Context) (int, error) {
	key := tokenBudgetKeyPrefix + "daily:" + time.Now().Format("20060102")
	val, err := b.redis.Get(ctx, key)
	if err != nil {
		return 0, nil // key 不存在返回 0
	}
	var used int
	fmt.Sscanf(val, "%d", &used)
	return used, nil
}

func (b *TokenBudget) getMonthlyUsed(ctx context.Context) (int, error) {
	key := tokenBudgetKeyPrefix + "monthly:" + time.Now().Format("200601")
	val, err := b.redis.Get(ctx, key)
	if err != nil {
		return 0, nil
	}
	var used int
	fmt.Sscanf(val, "%d", &used)
	return used, nil
}

func (b *TokenBudget) getUserUsed(ctx context.Context, userID string) (int, error) {
	key := tokenUserKeyPrefix + userID + ":" + time.Now().Format("200601")
	val, err := b.redis.Get(ctx, key)
	if err != nil {
		return 0, nil
	}
	var used int
	fmt.Sscanf(val, "%d", &used)
	return used, nil
}

func (b *TokenBudget) incrementDaily(ctx context.Context, tokens int) error {
	key := tokenBudgetKeyPrefix + "daily:" + time.Now().Format("20060102")
	// 简化实现：直接设置（生产环境应使用 INCRBY）
	current, _ := b.getDailyUsed(ctx)
	return b.redis.Set(ctx, key, fmt.Sprintf("%d", current+tokens), 25*time.Hour)
}

func (b *TokenBudget) incrementMonthly(ctx context.Context, tokens int) error {
	key := tokenBudgetKeyPrefix + "monthly:" + time.Now().Format("200601")
	current, _ := b.getMonthlyUsed(ctx)
	return b.redis.Set(ctx, key, fmt.Sprintf("%d", current+tokens), 32*24*time.Hour)
}

func (b *TokenBudget) incrementUser(ctx context.Context, userID string, tokens int) error {
	key := tokenUserKeyPrefix + userID + ":" + time.Now().Format("200601")
	current, _ := b.getUserUsed(ctx, userID)
	return b.redis.Set(ctx, key, fmt.Sprintf("%d", current+tokens), 32*24*time.Hour)
}

func (b *TokenBudget) checkThreshold(ctx context.Context, scope string, used, budget int, userID string) {
	threshold := budget * 80 / 100
	if used >= threshold && used < budget {
		logger.Warn("Token 预算接近上限",
			zap.String("scope", scope),
			zap.Int("used", used),
			zap.Int("budget", budget),
			zap.Int("threshold_pct", 80),
			zap.String("user_id", userID),
		)
	}
}

func (b *TokenBudget) recordBudgetExceeded(ctx context.Context, scope, userID string, used, budget int) {
	logger.Error("Token 预算已耗尽",
		zap.String("scope", scope),
		zap.Int("used", used),
		zap.Int("budget", budget),
		zap.String("user_id", userID),
	)

	if b.auditLog != nil {
		_ = b.auditLog.Record(ctx, audit.Entry{
			Action:   audit.ActionLLMDegraded,
			Level:    audit.LevelError,
			Module:   "agent",
			EntityID: userID,
			Detail: map[string]any{
				"scope":   scope,
				"used":    used,
				"budget":  budget,
				"reason":  fmt.Sprintf("%s Token 预算已耗尽", scope),
			},
		})
	}
}
