package database

import (
	"context"

	"gorm.io/gorm"
)

// Repository 通用 Repository 接口
// 所有业务模块通过此接口操作数据库，禁止直连 gorm.DB
// 禁止字符串拼 SQL，统一使用 GORM 参数化查询
type Repository[T any] interface {
	// Create 创建记录
	Create(ctx context.Context, entity *T) error
	// GetByID 根据 ID 查询
	GetByID(ctx context.Context, id string) (*T, error)
	// Update 更新记录
	Update(ctx context.Context, entity *T) error
	// Delete 软删除
	Delete(ctx context.Context, id string) error
	// List 分页查询
	List(ctx context.Context, opts ...QueryOption) ([]T, int64, error)
	// WithTx 在事务中执行
	WithTx(ctx context.Context, fn func(tx *gorm.DB) error) error
}

// QueryOption 查询选项
type QueryOption func(*QueryConfig)

// QueryConfig 查询配置
type QueryConfig struct {
	Conditions []Condition
	OrderBy    string
	PageNum    int
	PageSize   int
	Preloads   []string
}

// Condition 查询条件（参数化，禁止拼接 SQL）
type Condition struct {
	Query string        // 参数化查询片段，如 "status = ?"
	Args  []interface{} // 参数值
}

// WithCondition 添加查询条件
func WithCondition(query string, args ...interface{}) QueryOption {
	return func(c *QueryConfig) {
		c.Conditions = append(c.Conditions, Condition{Query: query, Args: args})
	}
}

// WithOrderBy 设置排序
func WithOrderBy(order string) QueryOption {
	return func(c *QueryConfig) {
		c.OrderBy = order
	}
}

// WithPagination 设置分页
func WithPagination(pageNum, pageSize int) QueryOption {
	return func(c *QueryConfig) {
		c.PageNum = pageNum
		c.PageSize = pageSize
	}
}

// WithPreload 预加载关联
func WithPreload(field string) QueryOption {
	return func(c *QueryConfig) {
		c.Preloads = append(c.Preloads, field)
	}
}

// genericRepository 通用 Repository 实现
type genericRepository[T any] struct {
	db *DB
}

// NewRepository 创建通用 Repository
// T 必须实现 TableName() 方法
func NewRepository[T any](db *DB) Repository[T] {
	return &genericRepository[T]{db: db}
}

func (r *genericRepository[T]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r *genericRepository[T]) GetByID(ctx context.Context, id string) (*T, error) {
	var entity T
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&entity).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *genericRepository[T]) Update(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

func (r *genericRepository[T]) Delete(ctx context.Context, id string) error {
	var entity T
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&entity).Error
}

func (r *genericRepository[T]) List(ctx context.Context, opts ...QueryOption) ([]T, int64, error) {
	cfg := &QueryConfig{
		PageNum:  1,
		PageSize: 20,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	var entity T
	query := r.db.WithContext(ctx).Model(&entity)

	// 应用预加载
	for _, preload := range cfg.Preloads {
		query = query.Preload(preload)
	}

	// 应用条件（参数化查询，禁止拼接 SQL）
	for _, cond := range cfg.Conditions {
		query = query.Where(cond.Query, cond.Args...)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序
	if cfg.OrderBy != "" {
		query = query.Order(cfg.OrderBy)
	}

	// 分页
	if cfg.PageNum > 0 && cfg.PageSize > 0 {
		offset := (cfg.PageNum - 1) * cfg.PageSize
		query = query.Offset(offset).Limit(cfg.PageSize)
	}

	var results []T
	if err := query.Find(&results).Error; err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

func (r *genericRepository[T]) WithTx(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}
