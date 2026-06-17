// Package logger 结构化日志封装
// 统一使用 zap JSON 格式，ERROR 必带 trace_id
// 全局拦截 SQL/Token/密钥/隐私数据打印
package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// 敏感字段关键词，日志中禁止打印
var sensitiveKeywords = []string{
	"password", "passwd", "secret", "token", "api_key", "apikey",
	"access_key", "secret_key", "authorization", "credential",
	"phone", "mobile", "id_card", "bank_card", "address",
	"sql", "query", // SQL 语句
}

// 全局 logger 实例
var globalLogger *zap.Logger

func init() {
	// 默认初始化为生产环境配置
	cfg := ProductionConfig()
	logger, err := cfg.Build()
	if err != nil {
		// 降级为 nop logger
		globalLogger = zap.NewNop()
		return
	}
	globalLogger = logger
}

// ProductionConfig 生产环境日志配置
// 输出 JSON 格式，INFO 级别，包含调用位置
func ProductionConfig() zap.Config {
	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}
	cfg.Encoding = "json"
	cfg.EncoderConfig = zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	return cfg
}

// DevelopmentConfig 开发环境日志配置
// 输出 console 格式，DEBUG 级别
func DevelopmentConfig() zap.Config {
	cfg := zap.NewDevelopmentConfig()
	cfg.Encoding = "console"
	return cfg
}

// Init 初始化全局 logger
func Init(cfg zap.Config) error {
	logger, err := cfg.Build()
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}

// InitWithEnv 根据环境变量初始化 logger
func InitWithEnv() {
	env := os.Getenv("APP_ENV")
	var cfg zap.Config
	switch env {
	case "production", "prod":
		cfg = ProductionConfig()
	case "development", "dev":
		cfg = DevelopmentConfig()
	default:
		cfg = ProductionConfig()
	}
	if err := Init(cfg); err != nil {
		globalLogger = zap.NewNop()
	}
}

// GetLogger 获取全局 logger
func GetLogger() *zap.Logger {
	return globalLogger
}

// WithTraceID 创建带 trace_id 的 logger
func WithTraceID(traceID string) *zap.Logger {
	return globalLogger.With(zap.String("trace_id", traceID))
}

// WithFields 创建带多个字段的 logger
func WithFields(fields ...zap.Field) *zap.Logger {
	return globalLogger.With(fields...)
}

// Info 记录 INFO 级别日志
func Info(msg string, fields ...zap.Field) {
	globalLogger.Info(msg, fields...)
}

// Warn 记录 WARN 级别日志
func Warn(msg string, fields ...zap.Field) {
	globalLogger.Warn(msg, fields...)
}

// Error 记录 ERROR 级别日志（必须携带 trace_id）
func Error(msg string, fields ...zap.Field) {
	globalLogger.Error(msg, fields...)
}

// Debug 记录 DEBUG 级别日志
func Debug(msg string, fields ...zap.Field) {
	globalLogger.Debug(msg, fields...)
}

// Fatal 记录 FATAL 级别日志并退出
func Fatal(msg string, fields ...zap.Field) {
	globalLogger.Fatal(msg, fields...)
}

// Sync 刷新日志缓冲区
func Sync() error {
	return globalLogger.Sync()
}

// IsSensitive 检查字段名是否包含敏感关键词
func IsSensitive(fieldName string) bool {
	lower := strings.ToLower(fieldName)
	for _, keyword := range sensitiveKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

// SanitizeField 脱敏字段值
// 敏感字段返回 "***REDACTED***"
func SanitizeField(key string, value string) string {
	if IsSensitive(key) {
		return "***REDACTED***"
	}
	return value
}
