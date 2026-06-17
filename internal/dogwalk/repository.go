package dogwalk

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/patch-pet/patch-pet/internal/database"
)

// Repository 代遛模块 Repository
type Repository struct {
	db *database.DB
}

// NewRepository 创建代遛 Repository
func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// CreateOpportunity 创建代遛需求
func (r *Repository) CreateOpportunity(ctx context.Context, opp *DogWalkOpportunity) error {
	return r.db.WithContext(ctx).Create(opp).Error
}

// GetOpportunityByID 根据 ID 获取需求
func (r *Repository) GetOpportunityByID(ctx context.Context, id string) (*DogWalkOpportunity, error) {
	var opp DogWalkOpportunity
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&opp).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("查询需求失败: %w", err)
	}
	return &opp, nil
}

// UpdateOpportunity 更新需求
func (r *Repository) UpdateOpportunity(ctx context.Context, opp *DogWalkOpportunity) error {
	return r.db.WithContext(ctx).Save(opp).Error
}

// CreatePlan 创建代遛方案
func (r *Repository) CreatePlan(ctx context.Context, plan *DogWalkPlan) error {
	return r.db.WithContext(ctx).Create(plan).Error
}

// GetPlanByID 根据 ID 获取方案
func (r *Repository) GetPlanByID(ctx context.Context, id string) (*DogWalkPlan, error) {
	var plan DogWalkPlan
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&plan).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("查询方案失败: %w", err)
	}
	return &plan, nil
}

// UpdatePlan 更新方案
func (r *Repository) UpdatePlan(ctx context.Context, plan *DogWalkPlan) error {
	return r.db.WithContext(ctx).Save(plan).Error
}

// CreateOrder 创建代遛订单
func (r *Repository) CreateOrder(ctx context.Context, order *DogWalkOrder) error {
	return r.db.WithContext(ctx).Create(order).Error
}

// GetOrderByID 根据 ID 获取订单
func (r *Repository) GetOrderByID(ctx context.Context, id string) (*DogWalkOrder, error) {
	var order DogWalkOrder
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&order).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("查询订单失败: %w", err)
	}
	return &order, nil
}

// UpdateOrder 更新订单
func (r *Repository) UpdateOrder(ctx context.Context, order *DogWalkOrder) error {
	return r.db.WithContext(ctx).Save(order).Error
}

// ListOrders 分页查询历史订单
func (r *Repository) ListOrders(ctx context.Context, userID string, pageNum, pageSize int) ([]DogWalkOrder, int64, error) {
	var orders []DogWalkOrder
	var total int64

	query := r.db.WithContext(ctx).Model(&DogWalkOrder{}).
		Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计订单数失败: %w", err)
	}

	offset := (pageNum - 1) * pageSize
	if err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&orders).Error; err != nil {
		return nil, 0, fmt.Errorf("查询订单列表失败: %w", err)
	}

	return orders, total, nil
}

// CreateReport 创建服务报告
func (r *Repository) CreateReport(ctx context.Context, report *DogWalkReport) error {
	return r.db.WithContext(ctx).Create(report).Error
}

// GetReportByOrderID 根据订单 ID 获取报告
func (r *Repository) GetReportByOrderID(ctx context.Context, orderID string) (*DogWalkReport, error) {
	var report DogWalkReport
	err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		First(&report).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("查询报告失败: %w", err)
	}
	return &report, nil
}

// CreateLiveEvent 创建实时事件
func (r *Repository) CreateLiveEvent(ctx context.Context, event *DogWalkLiveEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}
