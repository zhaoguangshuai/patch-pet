package database

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/patch-pet/patch-pet/pkg/logger"
)

// AutoMigrate 自动迁移表结构
// 仅用于开发/测试环境，生产环境必须使用迁移脚本
// 禁止在生产环境调用此方法
func AutoMigrate(db *gorm.DB, models ...interface{}) error {
	for _, model := range models {
		if err := db.AutoMigrate(model); err != nil {
			logger.Error("自动迁移失败",
				zap.String("model", fmt.Sprintf("%T", model)),
				zap.Error(err),
			)
			return fmt.Errorf("迁移 %T 失败: %w", model, err)
		}
	}
	logger.Info("数据库迁移完成", zap.Int("models", len(models)))
	return nil
}

// RegisterModels 注册所有业务模型（用于自动迁移）
// 仅在开发/测试环境使用，生产环境使用 migration 脚本
func RegisterModels() []interface{} {
	// 医疗模块 5 张表
	// medical_episode, care_task, care_task_action, medical_summary, medical_authorization
	// 代遛模块 5 张表
	// dog_walk_opportunity, dog_walk_plan, dog_walk_order, dog_walk_live_event, dog_walk_report
	// 具体模型在各业务包中定义，此处返回空切片
	// 调用方自行传入需要迁移的模型
	return nil
}
