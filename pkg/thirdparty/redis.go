// Package thirdparty 第三方 SDK 统一封装
// Redis 客户端：会话记忆、幂等键、分布式锁、热点缓存
// Key 命名格式：模块名:实体名:唯一标识
// 禁止硬编码过期时间，统一走常量/配置
package thirdparty

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/patch-pet/patch-pet/pkg/logger"
)

// RedisConfig Redis 连接配置
// 所有值从环境变量注入，禁止硬编码
type RedisConfig struct {
	Addr     string // Redis 地址（环境变量 REDIS_ADDR）
	Password string // Redis 密码（环境变量 REDIS_PASSWORD）
	DB       int    // 数据库编号
}

// DefaultRedisConfig 从环境变量加载 Redis 配置
func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		Addr:     getEnvOrDefault("REDIS_ADDR", "localhost:6379"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// RedisClient Redis 客户端封装
type RedisClient struct {
	client *redis.Client
	config RedisConfig
}

// NewRedisClient 创建 Redis 客户端
func NewRedisClient(cfg RedisConfig) *RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	return &RedisClient{
		client: rdb,
		config: cfg,
	}
}

// Init 初始化连接并验证可用性
func (r *RedisClient) Init(ctx context.Context) error {
	if err := r.client.Ping(ctx).Err(); err != nil {
		logger.Error("Redis 连接失败",
			zap.String("addr", r.config.Addr),
			zap.Error(err),
		)
		return fmt.Errorf("Redis 连接失败: %w", err)
	}
	logger.Info("Redis 连接成功", zap.String("addr", r.config.Addr))
	return nil
}

// Close 关闭连接
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Ping 健康检查
func (r *RedisClient) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// --- 基础操作 ---

// Get 获取值
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// Set 设置值（带过期时间）
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

// Del 删除键
func (r *RedisClient) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, key).Result()
	return n > 0, err
}

// SetNX 仅当键不存在时设置（用于幂等键）
func (r *RedisClient) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	return r.client.SetNX(ctx, key, value, ttl).Result()
}

// Incr 原子自增（用于限流计数）
func (r *RedisClient) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

// Expire 设置键过期时间
func (r *RedisClient) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return r.client.Expire(ctx, key, ttl).Err()
}

// --- Hash 操作 ---

// HSet 设置 Hash 字段
func (r *RedisClient) HSet(ctx context.Context, key string, values ...interface{}) error {
	return r.client.HSet(ctx, key, values...).Err()
}

// HGet 获取 Hash 字段
func (r *RedisClient) HGet(ctx context.Context, key, field string) (string, error) {
	return r.client.HGet(ctx, key, field).Result()
}

// HGetAll 获取 Hash 所有字段
func (r *RedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.client.HGetAll(ctx, key).Result()
}

// HDel 删除 Hash 字段
func (r *RedisClient) HDel(ctx context.Context, key string, fields ...string) error {
	return r.client.HDel(ctx, key, fields...).Err()
}

// --- 分布式锁 ---

// Lock 分布式锁
// 使用 SET NX EX 实现，防死锁
type Lock struct {
	client   *redis.Client
	key      string
	value    string
	ttl      time.Duration
	acquired bool
}

// AcquireLock 获取分布式锁
// key: 锁键名（格式：lock:模块:资源标识）
// ttl: 锁自动过期时间（防死锁）
// value: 锁持有者标识（建议使用 trace_id）
func (r *RedisClient) AcquireLock(ctx context.Context, key, value string, ttl time.Duration) (*Lock, error) {
	lockKey := fmt.Sprintf("lock:%s", key)
	ok, err := r.client.SetNX(ctx, lockKey, value, ttl).Result()
	if err != nil {
		return nil, fmt.Errorf("获取锁失败: %w", err)
	}

	l := &Lock{
		client:   r.client,
		key:      lockKey,
		value:    value,
		ttl:      ttl,
		acquired: ok,
	}

	if !ok {
		logger.Warn("锁获取失败（已被占用）",
			zap.String("key", lockKey),
			zap.String("holder", value),
		)
	}

	return l, nil
}

// Acquired 是否成功获取锁
func (l *Lock) Acquired() bool {
	return l.acquired
}

// Release 释放锁（仅释放自己持有的锁，Lua 脚本保证原子性）
func (l *Lock) Release(ctx context.Context) error {
	if !l.acquired {
		return nil
	}

	// Lua 脚本：仅当值匹配时删除（防误释放他人锁）
	script := redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`)

	_, err := script.Run(ctx, l.client, []string{l.key}, l.value).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("释放锁失败: %w", err)
	}

	l.acquired = false
	return nil
}
