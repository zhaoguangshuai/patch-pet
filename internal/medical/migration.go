package medical

import "gorm.io/gorm"

// Models 返回医疗模块所有需要迁移的模型
// 仅用于开发/测试环境 AutoMigrate，生产环境使用 migration 脚本
func Models() []interface{} {
	return []interface{}{
		&MedicalEpisode{},
		&CareTask{},
		&CareTaskAction{},
		&MedicalSummary{},
		&MedicalAuthorization{},
	}
}

// Migrate 执行医疗模块数据库迁移
// 仅开发/测试环境使用
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(Models()...)
}
