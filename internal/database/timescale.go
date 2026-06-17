package database

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/patch-pet/patch-pet/pkg/logger"
)

const (
	// TimescaleDB 默认分区保留天数
	DefaultRetentionDays = 90
	// TimescaleDB 默认分区间隔
	DefaultChunkInterval = "1 day"
)

// TimescaleConfig TimescaleDB 配置
type TimescaleConfig struct {
	RetentionDays int    // 分区保留天数，默认 90
	ChunkInterval string // 分区间隔，默认 1 day
}

// DefaultTimescaleConfig 返回默认 TimescaleDB 配置
func DefaultTimescaleConfig() TimescaleConfig {
	return TimescaleConfig{
		RetentionDays: DefaultRetentionDays,
		ChunkInterval: DefaultChunkInterval,
	}
}

// TimescaleDB TimescaleDB 封装
type TimescaleDB struct {
	db     *gorm.DB
	config TimescaleConfig
}

// NewTimescaleDB 创建 TimescaleDB 实例
// 复用 PostgreSQL 连接（TimescaleDB 是 PG 扩展）
func NewTimescaleDB(db *gorm.DB, cfg TimescaleConfig) *TimescaleDB {
	return &TimescaleDB{db: db, config: cfg}
}

// CreateHypertable 将普通表转换为 TimescaleDB hypertable
// timeColumn: 时间列名（如 created_at）
// chunkInterval: 分区间隔（如 "1 day"）
func (t *TimescaleDB) CreateHypertable(ctx context.Context, tableName, timeColumn string) error {
	sql := fmt.Sprintf(
		"SELECT create_hypertable('%s', '%s', chunk_time_interval => INTERVAL '%s', if_not_exists => true)",
		tableName, timeColumn, t.config.ChunkInterval,
	)

	if err := t.db.WithContext(ctx).Exec(sql).Error; err != nil {
		logger.Error("创建 hypertable 失败",
			zap.String("table", tableName),
			zap.Error(err),
		)
		return fmt.Errorf("创建 hypertable %s 失败: %w", tableName, err)
	}

	logger.Info("hypertable 创建成功",
		zap.String("table", tableName),
		zap.String("chunk_interval", t.config.ChunkInterval),
	)
	return nil
}

// SetRetentionPolicy 设置数据保留策略
// 超过 retentionDays 的数据自动删除
func (t *TimescaleDB) SetRetentionPolicy(ctx context.Context, tableName string, retentionDays int) error {
	sql := fmt.Sprintf(
		"SELECT add_retention_policy('%s', INTERVAL '%d days', if_not_exists => true)",
		tableName, retentionDays,
	)

	if err := t.db.WithContext(ctx).Exec(sql).Error; err != nil {
		logger.Error("设置保留策略失败",
			zap.String("table", tableName),
			zap.Int("retention_days", retentionDays),
			zap.Error(err),
		)
		return fmt.Errorf("设置 %s 保留策略失败: %w", tableName, err)
	}

	logger.Info("保留策略设置成功",
		zap.String("table", tableName),
		zap.Int("retention_days", retentionDays),
	)
	return nil
}

// DropRetentionPolicy 删除数据保留策略
func (t *TimescaleDB) DropRetentionPolicy(ctx context.Context, tableName string) error {
	sql := fmt.Sprintf(
		"SELECT remove_retention_policy('%s', if_exists => true)",
		tableName,
	)

	if err := t.db.WithContext(ctx).Exec(sql).Error; err != nil {
		return fmt.Errorf("删除 %s 保留策略失败: %w", tableName, err)
	}
	return nil
}

// QueryByTimeRange 按时间范围查询时序数据
// 返回指定时间范围内的记录，按时间正序
func (t *TimescaleDB) QueryByTimeRange(ctx context.Context, dest interface{}, tableName string, start, end time.Time, additionalWhere string, args ...interface{}) error {
	query := fmt.Sprintf(
		"SELECT * FROM %s WHERE created_at >= ? AND created_at < ?",
		tableName,
	)

	params := []interface{}{start, end}
	if additionalWhere != "" {
		query += " AND " + additionalWhere
		params = append(params, args...)
	}
	query += " ORDER BY created_at ASC"

	if err := t.db.WithContext(ctx).Raw(query, params...).Scan(dest).Error; err != nil {
		return fmt.Errorf("时序查询失败: %w", err)
	}
	return nil
}

// InitTimescaleModule 初始化 TimescaleDB 模块
// 创建 hypertable 并设置保留策略
func InitTimescaleModule(ctx context.Context, db *gorm.DB, cfg TimescaleConfig) (*TimescaleDB, error) {
	ts := NewTimescaleDB(db, cfg)

	// 检查 TimescaleDB 扩展是否已安装
	var installed bool
	err := db.WithContext(ctx).Raw(
		"SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'timescaledb')",
	).Scan(&installed).Error
	if err != nil {
		return nil, fmt.Errorf("检查 TimescaleDB 扩展失败: %w", err)
	}

	if !installed {
		logger.Warn("TimescaleDB 扩展未安装，时序功能不可用")
		return ts, nil
	}

	logger.Info("TimescaleDB 扩展已就绪")
	return ts, nil
}
