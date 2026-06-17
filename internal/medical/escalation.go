// Package medical 异常规则校验链
// Missed → EscalationCheck → SuggestContactClinic
// 当医疗任务未按时完成时，根据风险等级和频率决定升级策略
package medical

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/patch-pet/patch-pet/internal/audit"
	"github.com/patch-pet/patch-pet/internal/lifeflow"
	"github.com/patch-pet/patch-pet/pkg/logger"
)

// EscalationLevel 升级等级
type EscalationLevel string

const (
	EscalationNone     EscalationLevel = "none"      // 无需升级
	EscalationReminder EscalationLevel = "reminder"  // 提醒主人
	EscalationWarn     EscalationLevel = "warn"      // 警告，建议联系诊所
	EscalationCritical EscalationLevel = "critical"  // 紧急，强烈建议就医
)

// EscalationResult 升级检查结果
type EscalationResult struct {
	Level              EscalationLevel `json:"level"`                // 升级等级
	TaskID             string          `json:"task_id"`              // 任务 ID
	EpisodeID          string          `json:"episode_id"`           // 疗程 ID
	RiskLevel          string          `json:"risk_level"`           // 任务风险等级
	MissedCount        int             `json:"missed_count"`         // 连续未完成次数
	SuggestClinic      bool            `json:"suggest_clinic"`       // 是否建议联系诊所
	SuggestionMessage  string          `json:"suggestion_message"`   // 建议消息
}

// EscalationRule 升级规则定义
type EscalationRule struct {
	MinMissedCount int             // 最少未完成次数
	RiskLevels     []string        // 适用风险等级
	Level          EscalationLevel // 升级等级
	SuggestClinic  bool            // 是否建议联系诊所
}

// 默认升级规则链（按优先级从低到高）
var defaultEscalationRules = []EscalationRule{
	{
		MinMissedCount: 1,
		RiskLevels:     []string{"P0", "P1"},
		Level:          EscalationWarn,
		SuggestClinic:  true,
	},
	{
		MinMissedCount: 3,
		RiskLevels:     []string{"P2", "P3"},
		Level:          EscalationReminder,
		SuggestClinic:  false,
	},
	{
		MinMissedCount: 2,
		RiskLevels:     []string{"P0"},
		Level:          EscalationCritical,
		SuggestClinic:  true,
	},
	{
		MinMissedCount: 3,
		RiskLevels:     []string{"P1"},
		Level:          EscalationCritical,
		SuggestClinic:  true,
	},
}

// EscalationChain 异常规则校验链
type EscalationChain struct {
	repo     *Repository
	lifeflow lifeflow.Writer
	auditLog audit.Recorder
	rules    []EscalationRule
}

// NewEscalationChain 创建升级校验链
func NewEscalationChain(repo *Repository, lf lifeflow.Writer, ar audit.Recorder) *EscalationChain {
	return &EscalationChain{
		repo:     repo,
		lifeflow: lf,
		auditLog: ar,
		rules:    defaultEscalationRules,
	}
}

// ProcessMissedTask 处理未完成任务
// 执行链路：记录 Missed → EscalationCheck → SuggestContactClinic
func (c *EscalationChain) ProcessMissedTask(ctx context.Context, task CareTask) (*EscalationResult, error) {
	// Step 1: 记录任务未完成
	logger.Info("任务未完成",
		zap.String("task_id", task.ID),
		zap.String("episode_id", task.EpisodeID),
		zap.String("risk_level", task.RiskLevel),
	)

	// Step 2: 查询该疗程下同风险等级的未完成任务数
	missedCount, err := c.countMissedTasks(ctx, task.EpisodeID, task.RiskLevel)
	if err != nil {
		return nil, fmt.Errorf("统计未完成任务数失败: %w", err)
	}

	// Step 3: 执行升级检查
	result := c.evaluateEscalation(task, missedCount)

	// Step 4: 写 lifeflow 事件
	_ = lifeflow.WriteSafetyAlert(ctx, c.lifeflow, task.ID, "medical",
		"task_missed",
		fmt.Sprintf("任务未完成: task_id=%s, risk_level=%s, missed_count=%d, escalation=%s",
			task.ID, task.RiskLevel, missedCount, result.Level))

	// Step 5: 如果需要升级，写审计日志
	if result.Level != EscalationNone {
		_ = c.auditLog.Record(ctx, audit.Entry{
			Action:   audit.ActionHITLTrigger,
			Level:    audit.LevelWarn,
			Module:   "medical",
			EntityID: task.ID,
			Detail: map[string]any{
				"episode_id":    task.EpisodeID,
				"risk_level":    task.RiskLevel,
				"missed_count":  missedCount,
				"escalation":    string(result.Level),
				"suggest_clinic": result.SuggestClinic,
			},
		})
	}

	return result, nil
}

// countMissedTasks 统计疗程下指定风险等级的未完成任务数
func (c *EscalationChain) countMissedTasks(ctx context.Context, episodeID, riskLevel string) (int, error) {
	tasks, err := c.repo.GetTasksByEpisodeID(ctx, episodeID)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, t := range tasks {
		if t.RiskLevel == riskLevel && t.Status == "Missed" {
			count++
		}
	}
	return count, nil
}

// evaluateEscalation 根据规则链评估升级等级
func (c *EscalationChain) evaluateEscalation(task CareTask, missedCount int) *EscalationResult {
	result := &EscalationResult{
		TaskID:      task.ID,
		EpisodeID:   task.EpisodeID,
		RiskLevel:   task.RiskLevel,
		MissedCount: missedCount,
		Level:       EscalationNone,
	}

	// 按规则优先级匹配（最后匹配的规则生效，因为规则按从低到高排列）
	for _, rule := range c.rules {
		if missedCount >= rule.MinMissedCount && c.matchesRiskLevel(task.RiskLevel, rule.RiskLevels) {
			result.Level = rule.Level
			result.SuggestClinic = rule.SuggestClinic
		}
	}

	// 生成建议消息
	switch result.Level {
	case EscalationReminder:
		result.SuggestionMessage = fmt.Sprintf("任务 %s 已未完成 %d 次，请注意按时完成", task.ID, missedCount)
	case EscalationWarn:
		result.SuggestionMessage = fmt.Sprintf("任务 %s (风险等级: %s) 已未完成 %d 次，建议联系宠物诊所咨询", task.ID, task.RiskLevel, missedCount)
	case EscalationCritical:
		result.SuggestionMessage = fmt.Sprintf("紧急：任务 %s (风险等级: %s) 已未完成 %d 次，强烈建议立即联系宠物诊所", task.ID, task.RiskLevel, missedCount)
	}

	return result
}

// matchesRiskLevel 检查风险等级是否在列表中
func (c *EscalationChain) matchesRiskLevel(level string, levels []string) bool {
	for _, l := range levels {
		if l == level {
			return true
		}
	}
	return false
}
