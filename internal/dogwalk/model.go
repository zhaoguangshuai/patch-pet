// Package dogwalk 代遛狗模块（P1 中高危）
// 遛狗需求识别、服务商筛选、路线规划、订单履约、服务报告生成
// 禁止 AI 自动下单、强制营销
package dogwalk

import (
	"github.com/patch-pet/patch-pet/pkg/types"
	"gorm.io/gorm"
)

// OpportunityStatus 代遛需求状态
type OpportunityStatus string

const (
	OppCandidateDetected  OpportunityStatus = "CandidateDetected"
	OppAwaitingPermission OpportunityStatus = "AwaitingPermission"
	OppRejected           OpportunityStatus = "Rejected"
	OppReminderOnly       OpportunityStatus = "ReminderOnly"
)

// OrderStatus 代遛订单状态（14 个状态）
type OrderStatus string

const (
	OrderPlanDrafting      OrderStatus = "PlanDrafting"
	OrderVendorSelected    OrderStatus = "VendorSelected"
	OrderRoutePreference   OrderStatus = "RoutePreference"
	OrderRouteReady        OrderStatus = "RouteReady"
	OrderAwaitingPayment   OrderStatus = "AwaitingPayment"
	OrderPaid              OrderStatus = "Paid"
	OrderBooked            OrderStatus = "Booked"
	OrderInService         OrderStatus = "InService"
	OrderAnomalyDetected   OrderStatus = "AnomalyDetected"
	OrderCompleted         OrderStatus = "Completed"
	OrderReportReady       OrderStatus = "ReportReady"
	OrderLifeStreamWritten OrderStatus = "LifeStreamWritten"
	OrderCancelled         OrderStatus = "Cancelled"
	OrderRefunded          OrderStatus = "Refunded"
)

// DogWalkOpportunity 代遛需求实体
type DogWalkOpportunity struct {
	ID                string         `json:"id" gorm:"column:id;primaryKey;type:varchar(64)"`
	FamilyID          string         `json:"family_id" gorm:"column:family_id;type:varchar(64);not null;index:idx_dog_walk_opportunity_family_id"`
	PetID             string         `json:"pet_id" gorm:"column:pet_id;type:varchar(64);not null;index:idx_dog_walk_opportunity_pet_id"`
	TriggerReasonJSON string         `json:"trigger_reason_json" gorm:"column:trigger_reason_json;type:jsonb"`
	Confidence        float64        `json:"confidence" gorm:"column:confidence;type:decimal(4,2)"`
	Status            string         `json:"status" gorm:"column:status;type:varchar(32);not null;index:idx_dog_walk_opportunity_status"`
	CreatedBy         string         `json:"created_by" gorm:"column:created_by;type:varchar(64)"`
	CreatedAt         types.CSTTime  `json:"created_at" gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt         types.CSTTime  `json:"updated_at" gorm:"column:updated_at;type:timestamptz;not null"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"column:deleted_at;index:idx_dog_walk_opportunity_deleted_at"`
}

func (DogWalkOpportunity) TableName() string { return "dog_walk_opportunity" }

// DogWalkPlan 代遛方案实体
type DogWalkPlan struct {
	ID                   string         `json:"id" gorm:"column:id;primaryKey;type:varchar(64)"`
	OpportunityID        string         `json:"opportunity_id" gorm:"column:opportunity_id;type:varchar(64);not null;index:idx_dog_walk_plan_opportunity_id"`
	VendorID             string         `json:"vendor_id" gorm:"column:vendor_id;type:varchar(64);index:idx_dog_walk_plan_vendor_id"`
	RouteID              string         `json:"route_id" gorm:"column:route_id;type:varchar(64)"`
	RoutePreferenceJSON  string         `json:"route_preference_json" gorm:"column:route_preference_json;type:jsonb"`
	EstimatedDistanceM   int            `json:"estimated_distance_m" gorm:"column:estimated_distance_m"`
	EstimatedDurationMin int            `json:"estimated_duration_min" gorm:"column:estimated_duration_min"`
	PriceCents           int            `json:"price_cents" gorm:"column:price_cents"`
	Status               string         `json:"status" gorm:"column:status;type:varchar(32);not null;index:idx_dog_walk_plan_status"`
	CreatedBy            string         `json:"created_by" gorm:"column:created_by;type:varchar(64)"`
	CreatedAt            types.CSTTime  `json:"created_at" gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt            types.CSTTime  `json:"updated_at" gorm:"column:updated_at;type:timestamptz;not null"`
	DeletedAt            gorm.DeletedAt `json:"-" gorm:"column:deleted_at;index:idx_dog_walk_plan_deleted_at"`
}

func (DogWalkPlan) TableName() string { return "dog_walk_plan" }

// DogWalkOrder 代遛订单实体
type DogWalkOrder struct {
	ID              string         `json:"id" gorm:"column:id;primaryKey;type:varchar(64)"`
	PlanID          string         `json:"plan_id" gorm:"column:plan_id;type:varchar(64);not null;index:idx_dog_walk_order_plan_id"`
	UserID          string         `json:"user_id" gorm:"column:user_id;type:varchar(64);not null;index:idx_dog_walk_order_user_id"`
	PaymentOrderID  string         `json:"payment_order_id" gorm:"column:payment_order_id;type:varchar(64);uniqueIndex:idx_dog_walk_order_payment_order_id_deleted_at"`
	VendorOrderID   string         `json:"vendor_order_id" gorm:"column:vendor_order_id;type:varchar(64)"`
	Status          OrderStatus    `json:"status" gorm:"column:status;type:varchar(32);not null;index:idx_dog_walk_order_status"`
	IdempotencyKey  string         `json:"idempotency_key" gorm:"column:idempotency_key;type:varchar(64);not null;uniqueIndex:idx_dog_walk_order_idempotency_key_deleted_at"`
	CreatedBy       string         `json:"created_by" gorm:"column:created_by;type:varchar(64)"`
	PaidAt          types.NullableCSTTime `json:"paid_at" gorm:"column:paid_at;type:timestamptz"`
	BookedAt        types.NullableCSTTime `json:"booked_at" gorm:"column:booked_at;type:timestamptz"`
	StartedAt       types.NullableCSTTime `json:"started_at" gorm:"column:started_at;type:timestamptz"`
	CompletedAt     types.NullableCSTTime `json:"completed_at" gorm:"column:completed_at;type:timestamptz"`
	CreatedAt       types.CSTTime         `json:"created_at" gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt       types.CSTTime         `json:"updated_at" gorm:"column:updated_at;type:timestamptz;not null"`
	DeletedAt       gorm.DeletedAt        `json:"-" gorm:"column:deleted_at;index:idx_dog_walk_order_deleted_at"`
}

func (DogWalkOrder) TableName() string { return "dog_walk_order" }

// DogWalkLiveEvent 代遛实时事件（TimescaleDB hypertable）
type DogWalkLiveEvent struct {
	ID           string        `json:"id" gorm:"column:id;type:varchar(64)"`
	OrderID      string        `json:"order_id" gorm:"column:order_id;type:varchar(64);not null;index:idx_dog_walk_live_event_order_id"`
	EventType    string        `json:"event_type" gorm:"column:event_type;type:varchar(32);not null;index:idx_dog_walk_live_event_event_type"`
	LocationJSON string        `json:"location_json" gorm:"column:location_json;type:jsonb"`
	MetricsJSON  string        `json:"metrics_json" gorm:"column:metrics_json;type:jsonb"`
	RiskLevel    string        `json:"risk_level" gorm:"column:risk_level;type:varchar(8)"`
	CreatedBy    string        `json:"created_by" gorm:"column:created_by;type:varchar(64)"`
	CreatedAt    types.CSTTime `json:"created_at" gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt    types.CSTTime `json:"updated_at" gorm:"column:updated_at;type:timestamptz;not null"`
}

func (DogWalkLiveEvent) TableName() string { return "dog_walk_live_event" }

// DogWalkReport 代遛服务报告实体
type DogWalkReport struct {
	ID                string         `json:"id" gorm:"column:id;primaryKey;type:varchar(64)"`
	OrderID           string         `json:"order_id" gorm:"column:order_id;type:varchar(64);not null;index:idx_dog_walk_report_order_id"`
	RouteSnapshotJSON string         `json:"route_snapshot_json" gorm:"column:route_snapshot_json;type:jsonb"`
	PhotosJSON        string         `json:"photos_json" gorm:"column:photos_json;type:jsonb"`
	BehaviorSummary   string         `json:"behavior_summary" gorm:"column:behavior_summary;type:text"`
	VendorFeedback    string         `json:"vendor_feedback" gorm:"column:vendor_feedback;type:text"`
	OwnerRating       int            `json:"owner_rating" gorm:"column:owner_rating"`
	CreatedBy         string         `json:"created_by" gorm:"column:created_by;type:varchar(64)"`
	CreatedAt         types.CSTTime  `json:"created_at" gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt         types.CSTTime  `json:"updated_at" gorm:"column:updated_at;type:timestamptz;not null"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"column:deleted_at;index:idx_dog_walk_report_deleted_at"`
}

func (DogWalkReport) TableName() string { return "dog_walk_report" }
