// Package medical 医疗任务定时调度
// 分布式锁防重，超时阈值 10s，仅查询/推送/提醒，禁改业务数据
package medical

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/patch-pet/patch-pet/internal/audit"
	"github.com/patch-pet/patch-pet/internal/lifeflow"
	"github.com/patch-pet/patch-pet/pkg/logger"
	"github.com/patch-pet/patch-pet/pkg/thirdparty"
)

const (
	// SchedulerLockKey 调度器分布式锁键
	SchedulerLockKey = "medical:scheduler:lock"
	// SchedulerLockTTL 锁自动过期时间（防死锁）
	SchedulerLockTTL = 30 * time.Second
	// SchedulerTimeout 单次调度执行超时
	SchedulerTimeout = 10 * time.Second
	// MaxTasksPerBatch 每批最大处理任务数
	MaxTasksPerBatch = 50
	// OverdueThresholdMinutes 超期阈值（分钟）
	OverdueThresholdMinutes = 30
)

// Scheduler 医疗任务调度器
// 仅执行查询/推送/提醒操作，禁止修改业务数据
type Scheduler struct {
	repo      *Repository
	redis     *thirdparty.RedisClient
	lifeflow  lifeflow.Writer
	auditLog  audit.Recorder
}

// NewScheduler 创建调度器
func NewScheduler(repo *Repository, redis *thirdparty.RedisClient, lf lifeflow.Writer, ar audit.Recorder) *Scheduler {
	return &Scheduler{
		repo:     repo,
		redis:    redis,
		lifeflow: lf,
		auditLog: ar,
	}
}

// Run 执行一次调度（由定时器触发）
// 1. 尝试获取分布式锁，防止多实例重复执行
// 2. 查询到期任务，发送提醒
// 3. 查询超期任务，触发升级告警
// 4. 释放锁
func (s *Scheduler) Run(ctx context.Context) {
	// 生成锁持有者标识（使用时间戳，便于排查）
	lockHolder := fmt.Sprintf("scheduler-%d", time.Now().UnixMilli())

	// 尝试获取分布式锁
	lock, err := s.redis.AcquireLock(ctx, SchedulerLockKey, lockHolder, SchedulerLockTTL)
	if err != nil {
		logger.Error("调度器获取锁失败", zap.Error(err))
		return
	}
	if !lock.Acquired() {
		logger.Debug("调度器锁已被占用，跳过本次执行")
		return
	}
	defer func() {
		if releaseErr := lock.Release(ctx); releaseErr != nil {
			logger.Error("调度器释放锁失败", zap.Error(releaseErr))
		}
	}()

	// 带超时的执行上下文
	execCtx, cancel := context.WithTimeout(ctx, SchedulerTimeout)
	defer cancel()

	// 1. 查询到期任务并发送提醒
	s.processDueTasks(execCtx)

	// 2. 查询超期任务并触发升级告警
	s.processOverdueTasks(execCtx)
}

// processDueTasks 处理到期任务：发送提醒通知
// 仅读取数据 + 写 lifeflow 事件，不修改任务状态
func (s *Scheduler) processDueTasks(ctx context.Context) {
	tasks, err := s.repo.GetDueTasks(ctx, MaxTasksPerBatch)
	if err != nil {
		logger.Error("查询到期任务失败", zap.Error(err))
		return
	}

	for _, task := range tasks {
		// 写 lifeflow 提醒事件（不修改业务数据）
		_ = lifeflow.WriteAgentAction(ctx, s.lifeflow, task.ID, "medical_scheduler", "task_due_reminder",
			map[string]any{
				"episode_id": task.EpisodeID,
				"status":     task.Status,
				"risk_level": task.RiskLevel,
				"message":    fmt.Sprintf("任务 %s (疗程 %s) 已到期", task.ID, task.EpisodeID),
			})

		logger.Info("到期任务提醒",
			zap.String("task_id", task.ID),
			zap.String("episode_id", task.EpisodeID),
			zap.String("status", task.Status),
			zap.String("risk_level", task.RiskLevel),
		)
	}
}

// processOverdueTasks 处理超期任务：触发升级告警
// 仅读取数据 + 写 lifeflow/审计事件，不修改任务状态
func (s *Scheduler) processOverdueTasks(ctx context.Context) {
	tasks, err := s.repo.GetOverdueTasks(ctx, OverdueThresholdMinutes, MaxTasksPerBatch)
	if err != nil {
		logger.Error("查询超期任务失败", zap.Error(err))
		return
	}

	for _, task := range tasks {
		// 写 lifeflow 安全告警
		_ = lifeflow.WriteSafetyAlert(ctx, s.lifeflow, task.ID, "medical_scheduler", "task_overdue",
			fmt.Sprintf("任务 %s (疗程 %s) 超期 %d 分钟未完成，状态: %s，风险等级: %s",
				task.ID, task.EpisodeID, OverdueThresholdMinutes, task.Status, task.RiskLevel))

		// 写审计日志
		_ = s.auditLog.Record(ctx, audit.Entry{
			Action:   audit.ActionHITLTrigger,
			Level:    audit.LevelWarn,
			Module:   "medical",
			EntityID: task.ID,
			Detail: map[string]any{
				"episode_id": task.EpisodeID,
				"risk_level": task.RiskLevel,
				"reason":     fmt.Sprintf("超期 %d 分钟未完成，触发人工介入", OverdueThresholdMinutes),
			},
		})

		logger.Warn("超期任务告警",
			zap.String("task_id", task.ID),
			zap.String("episode_id", task.EpisodeID),
			zap.String("status", task.Status),
			zap.String("risk_level", task.RiskLevel),
		)
	}
}
