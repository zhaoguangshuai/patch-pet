package medical

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/patch-pet/patch-pet/internal/database"
)

// Repository 医疗模块 Repository
type Repository struct {
	db *database.DB
}

// NewRepository 创建医疗 Repository
func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// GetCurrentEpisode 获取宠物当前疗程
// 当前疗程 = 最近一个非 Closed 状态的疗程
func (r *Repository) GetCurrentEpisode(ctx context.Context, petID string) (*MedicalEpisode, error) {
	var episode MedicalEpisode
	err := r.db.WithContext(ctx).
		Where("pet_id = ? AND status != ?", petID, EpisodeClosed).
		Order("created_at DESC").
		First(&episode).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("查询当前疗程失败: %w", err)
	}
	return &episode, nil
}

// GetEpisodeByID 根据 ID 获取疗程
func (r *Repository) GetEpisodeByID(ctx context.Context, episodeID string) (*MedicalEpisode, error) {
	var episode MedicalEpisode
	err := r.db.WithContext(ctx).
		Where("id = ?", episodeID).
		First(&episode).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("查询疗程失败: %w", err)
	}
	return &episode, nil
}

// ListEpisodes 分页查询历史疗程
func (r *Repository) ListEpisodes(ctx context.Context, petID string, pageNum, pageSize int) ([]MedicalEpisode, int64, error) {
	var episodes []MedicalEpisode
	var total int64

	query := r.db.WithContext(ctx).Model(&MedicalEpisode{}).
		Where("pet_id = ?", petID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计疗程数失败: %w", err)
	}

	offset := (pageNum - 1) * pageSize
	if err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&episodes).Error; err != nil {
		return nil, 0, fmt.Errorf("查询疗程列表失败: %w", err)
	}

	return episodes, total, nil
}

// GetTasksByEpisodeID 获取疗程下的所有任务
func (r *Repository) GetTasksByEpisodeID(ctx context.Context, episodeID string) ([]CareTask, error) {
	var tasks []CareTask
	err := r.db.WithContext(ctx).
		Where("episode_id = ?", episodeID).
		Order("due_at ASC").
		Find(&tasks).Error

	if err != nil {
		return nil, fmt.Errorf("查询护理任务失败: %w", err)
	}
	return tasks, nil
}

// CreateEpisode 创建疗程
func (r *Repository) CreateEpisode(ctx context.Context, episode *MedicalEpisode) error {
	return r.db.WithContext(ctx).Create(episode).Error
}

// UpdateEpisode 更新疗程
func (r *Repository) UpdateEpisode(ctx context.Context, episode *MedicalEpisode) error {
	return r.db.WithContext(ctx).Save(episode).Error
}

// CreateTasks 批量创建护理任务
func (r *Repository) CreateTasks(ctx context.Context, tasks []CareTask) error {
	if len(tasks) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&tasks).Error
}

// GetTaskByID 根据 ID 获取任务
func (r *Repository) GetTaskByID(ctx context.Context, taskID string) (*CareTask, error) {
	var task CareTask
	err := r.db.WithContext(ctx).
		Where("id = ?", taskID).
		First(&task).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}
	return &task, nil
}

// UpdateTask 更新任务
func (r *Repository) UpdateTask(ctx context.Context, task *CareTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

// CreateAction 创建任务动作记录
func (r *Repository) CreateAction(ctx context.Context, action *CareTaskAction) error {
	return r.db.WithContext(ctx).Create(action).Error
}

// CreateSummary 创建医疗摘要
func (r *Repository) CreateSummary(ctx context.Context, summary *MedicalSummary) error {
	return r.db.WithContext(ctx).Create(summary).Error
}

// GetSummaryByID 根据 ID 获取摘要
func (r *Repository) GetSummaryByID(ctx context.Context, summaryID string) (*MedicalSummary, error) {
	var summary MedicalSummary
	err := r.db.WithContext(ctx).
		Where("id = ?", summaryID).
		First(&summary).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("查询摘要失败: %w", err)
	}
	return &summary, nil
}

// GetActiveAuthorization 获取有效的医疗授权
// 有效授权 = 未撤销 且 未过期
func (r *Repository) GetActiveAuthorization(ctx context.Context, episodeID string) (*MedicalAuthorization, error) {
	var auth MedicalAuthorization
	err := r.db.WithContext(ctx).
		Where("episode_id = ? AND status = ? AND revoked_at IS NULL", episodeID, "active").
		First(&auth).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("查询授权失败: %w", err)
	}
	return &auth, nil
}

// CreateAuthorization 创建医疗授权
func (r *Repository) CreateAuthorization(ctx context.Context, auth *MedicalAuthorization) error {
	return r.db.WithContext(ctx).Create(auth).Error
}

// UpdateAuthorization 更新医疗授权
func (r *Repository) UpdateAuthorization(ctx context.Context, auth *MedicalAuthorization) error {
	return r.db.WithContext(ctx).Save(auth).Error
}

// GetDueTasks 查询到期未完成的任务
// 到期任务 = due_at <= now AND status IN (Active, Draft)
func (r *Repository) GetDueTasks(ctx context.Context, limit int) ([]CareTask, error) {
	var tasks []CareTask
	err := r.db.WithContext(ctx).
		Where("due_at <= NOW() AND status IN ?", []string{"Active", "Draft"}).
		Order("due_at ASC").
		Limit(limit).
		Find(&tasks).Error

	if err != nil {
		return nil, fmt.Errorf("查询到期任务失败: %w", err)
	}
	return tasks, nil
}

// GetOverdueTasks 查询超期未完成的任务
// 超期任务 = due_at < now - threshold AND status IN (Active, Draft, TaskDue)
func (r *Repository) GetOverdueTasks(ctx context.Context, thresholdMinutes int, limit int) ([]CareTask, error) {
	var tasks []CareTask
	err := r.db.WithContext(ctx).
		Where("due_at < NOW() - INTERVAL '? minutes' AND status IN ?",
			thresholdMinutes, []string{"Active", "Draft", "TaskDue"}).
		Order("due_at ASC").
		Limit(limit).
		Find(&tasks).Error

	if err != nil {
		return nil, fmt.Errorf("查询超期任务失败: %w", err)
	}
	return tasks, nil
}
