// Package database 数据库连接层
// 统一 GORM 初始化、连接池配置、健康检查
// 禁止业务直连数据库，统一走 Repository 层
// 禁止字符串拼 SQL，统一使用 GORM 参数化查询
package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"go.uber.org/zap"

	"github.com/patch-pet/patch-pet/pkg/logger"
)

// DBConfig 数据库连接配置
// 所有值从环境变量注入，禁止硬编码
type DBConfig struct {
	DSN             string // 数据库连接串（环境变量 POSTGRES_DSN）
	MaxOpenConns    int    // 最大打开连接数
	MaxIdleConns    int    // 最大空闲连接数
	ConnMaxLifetime int    // 连接最大存活时间（秒）
	ConnMaxIdleTime int    // 连接最大空闲时间（秒）
}

// DefaultDBConfig 返回默认数据库配置
func DefaultDBConfig() DBConfig {
	return DBConfig{
		DSN:             os.Getenv("POSTGRES_DSN"),
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 300, // 5 分钟
		ConnMaxIdleTime: 60,  // 1 分钟
	}
}

// DB 数据库实例封装
type DB struct {
	*gorm.DB
	config DBConfig
}

// New 创建数据库连接
// 连接失败不阻塞启动，返回错误供调用方决定降级策略
func New(cfg DBConfig) (*DB, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("POSTGRES_DSN 未配置")
	}

	// 自定义 GORM logger，桥接 zap
	gormLog := newGormLogger()

	gormCfg := &gorm.Config{
		Logger:                 gormLog,
		SkipDefaultTransaction: true, // 禁用默认事务，业务层按需开启
		PrepareStmt:            true, // 预编译语句，防 SQL 注入
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败: %w", err)
	}

	// 连接池配置
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层 sql.DB 失败: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTime) * time.Second)

	return &DB{DB: db, config: cfg}, nil
}

// HealthCheck 健康检查
func (d *DB) HealthCheck(ctx context.Context) error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("获取 sql.DB 失败: %w", err)
	}
	return sqlDB.PingContext(ctx)
}

// Close 关闭数据库连接
func (d *DB) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Stats 连接池统计信息
func (d *DB) Stats() map[string]interface{} {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return nil
	}
	stats := sqlDB.Stats()
	return map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration.String(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}
}

// gormLogger GORM 日志桥接器
// 将 GORM 日志输出桥接到 zap，拦截敏感 SQL 打印
type gormLogger struct {
	logger *zap.Logger
}

func newGormLogger() gormlogger.Interface {
	return &gormLogger{
		logger: logger.GetLogger().With(zap.String("component", "gorm")),
	}
}

func (l *gormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return l
}

func (l *gormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Info(msg, zap.Any("data", data))
}

func (l *gormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Warn(msg, zap.Any("data", data))
}

func (l *gormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Error(msg, zap.Any("data", data))
}

func (l *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	_, rows := fc()

	// 仅记录慢查询和错误，正常查询不打印 SQL（防泄露）
	if err != nil {
		l.logger.Error("SQL 执行失败",
			zap.Error(err),
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
		)
		return
	}

	if elapsed > 200*time.Millisecond {
		l.logger.Warn("慢查询",
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
		)
	}
}
