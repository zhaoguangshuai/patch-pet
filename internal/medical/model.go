// Package medical 医疗居家模块（P0 高危）
// 医嘱解析、护理任务调度、设备数据汇总、医疗摘要生成、授权管理、诊所对接
// 禁止 AI 执行医疗诊断
package medical

import (
	"github.com/patch-pet/patch-pet/pkg/types"
	"gorm.io/gorm"
)

// EpisodeStatus 医疗疗程状态（13 个状态）
type EpisodeStatus string

const (
	EpisodeCreated             EpisodeStatus = "EpisodeCreated"
	InstructionDrafted         EpisodeStatus = "InstructionDrafted"
	AwaitingOwnerConfirm       EpisodeStatus = "AwaitingOwnerConfirm"
	EpisodeActive              EpisodeStatus = "Active"
	TaskDue                    EpisodeStatus = "TaskDue"
	TaskDone                   EpisodeStatus = "Done"
	TaskSnoozed                EpisodeStatus = "Snoozed"
	TaskMissed                 EpisodeStatus = "Missed"
	SummaryReady               EpisodeStatus = "SummaryReady"
	AwaitingAuthorization      EpisodeStatus = "AwaitingAuthorization"
	SharedToClinic             EpisodeStatus = "SharedToClinic"
	HostedCare                 EpisodeStatus = "HostedCare"
	EpisodeClosed              EpisodeStatus = "Closed"
)

// MedicalEpisode 医疗疗程实体
type MedicalEpisode struct {
	ID               string                `json:"id" gorm:"column:id;primaryKey;type:varchar(64)"`
	FamilyID         string                `json:"family_id" gorm:"column:family_id;type:varchar(64);not null;index:idx_medical_episode_family_id"`
	PetID            string                `json:"pet_id" gorm:"column:pet_id;type:varchar(64);not null;index:idx_medical_episode_pet_id"`
	ClinicID         string                `json:"clinic_id" gorm:"column:clinic_id;type:varchar(64)"`
	Status           EpisodeStatus         `json:"status" gorm:"column:status;type:varchar(32);not null;index:idx_medical_episode_status"`
	CareMode         string                `json:"care_mode" gorm:"column:care_mode;type:varchar(32)"`
	StartedAt        types.NullableCSTTime `json:"started_at" gorm:"column:started_at;type:timestamptz"`
	ExpectedReviewAt types.NullableCSTTime `json:"expected_review_at" gorm:"column:expected_review_at;type:timestamptz"`
	SourceType       string                `json:"source_type" gorm:"column:source_type;type:varchar(32)"`
	SourceID         string                `json:"source_id" gorm:"column:source_id;type:varchar(64)"`
	CreatedBy        string                `json:"created_by" gorm:"column:created_by;type:varchar(64)"`
	CreatedAt        types.CSTTime         `json:"created_at" gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt        types.CSTTime         `json:"updated_at" gorm:"column:updated_at;type:timestamptz;not null"`
	DeletedAt        gorm.DeletedAt        `json:"-" gorm:"column:deleted_at;index:idx_medical_episode_deleted_at"`
}

func (MedicalEpisode) TableName() string { return "medical_episode" }

// CareTask 护理任务实体
type CareTask struct {
	ID               string                `json:"id" gorm:"column:id;primaryKey;type:varchar(64)"`
	EpisodeID        string                `json:"episode_id" gorm:"column:episode_id;type:varchar(64);not null;index:idx_care_task_episode_id"`
	TaskType         string                `json:"task_type" gorm:"column:task_type;type:varchar(32);not null"`
	Title            string                `json:"title" gorm:"column:title;type:varchar(255);not null"`
	Instruction      string                `json:"instruction" gorm:"column:instruction;type:text"`
	DueAt            types.NullableCSTTime `json:"due_at" gorm:"column:due_at;type:timestamptz;index:idx_care_task_due_at"`
	Status           string                `json:"status" gorm:"column:status;type:varchar(32);not null;index:idx_care_task_status"`
	SourceType       string                `json:"source_type" gorm:"column:source_type;type:varchar(32)"`
	SourceID         string                `json:"source_id" gorm:"column:source_id;type:varchar(64)"`
	LockedFieldsJSON string                `json:"locked_fields_json" gorm:"column:locked_fields_json;type:jsonb"`
	RiskLevel        string                `json:"risk_level" gorm:"column:risk_level;type:varchar(8)"`
	CreatedBy        string                `json:"created_by" gorm:"column:created_by;type:varchar(64)"`
	CreatedAt        types.CSTTime         `json:"created_at" gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt        types.CSTTime         `json:"updated_at" gorm:"column:updated_at;type:timestamptz;not null"`
	DeletedAt        gorm.DeletedAt        `json:"-" gorm:"column:deleted_at;index:idx_care_task_deleted_at"`
}

func (CareTask) TableName() string { return "care_task" }

// CareTaskAction 任务动作实体（含幂等键）
type CareTaskAction struct {
	ID              string         `json:"id" gorm:"column:id;primaryKey;type:varchar(64)"`
	TaskID          string         `json:"task_id" gorm:"column:task_id;type:varchar(64);not null;index:idx_care_task_action_task_id"`
	UserID          string         `json:"user_id" gorm:"column:user_id;type:varchar(64);not null"`
	ActionType      string         `json:"action_type" gorm:"column:action_type;type:varchar(32);not null"`
	Note            string         `json:"note" gorm:"column:note;type:text"`
	AttachmentsJSON string         `json:"attachments_json" gorm:"column:attachments_json;type:jsonb"`
	IdempotencyKey  string         `json:"idempotency_key" gorm:"column:idempotency_key;type:varchar(64);not null;uniqueIndex:idx_care_task_action_idempotency_key_deleted_at"`
	CreatedBy       string         `json:"created_by" gorm:"column:created_by;type:varchar(64)"`
	CreatedAt       types.CSTTime  `json:"created_at" gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt       types.CSTTime  `json:"updated_at" gorm:"column:updated_at;type:timestamptz;not null"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"column:deleted_at;index:idx_care_task_action_deleted_at"`
}

func (CareTaskAction) TableName() string { return "care_task_action" }

// MedicalSummary 医疗摘要实体
type MedicalSummary struct {
	ID              string                `json:"id" gorm:"column:id;primaryKey;type:varchar(64)"`
	EpisodeID       string                `json:"episode_id" gorm:"column:episode_id;type:varchar(64);not null;index:idx_medical_summary_episode_id"`
	WindowStart     types.NullableCSTTime `json:"window_start" gorm:"column:window_start;type:timestamptz"`
	WindowEnd       types.NullableCSTTime `json:"window_end" gorm:"column:window_end;type:timestamptz"`
	DataScopesJSON  string                `json:"data_scopes_json" gorm:"column:data_scopes_json;type:jsonb"`
	OwnerText       string                `json:"owner_text" gorm:"column:owner_text;type:text"`
	ClinicText      string                `json:"clinic_text" gorm:"column:clinic_text;type:text"`
	SafetyStatus    string                `json:"safety_status" gorm:"column:safety_status;type:varchar(32)"`
	GeneratedBy     string                `json:"generated_by" gorm:"column:generated_by;type:varchar(32)"`
	CreatedBy       string                `json:"created_by" gorm:"column:created_by;type:varchar(64)"`
	CreatedAt       types.CSTTime         `json:"created_at" gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt       types.CSTTime         `json:"updated_at" gorm:"column:updated_at;type:timestamptz;not null"`
	DeletedAt       gorm.DeletedAt        `json:"-" gorm:"column:deleted_at;index:idx_medical_summary_deleted_at"`
}

func (MedicalSummary) TableName() string { return "medical_summary" }

// MedicalAuthorization 医疗数据授权实体
type MedicalAuthorization struct {
	ID              string                `json:"id" gorm:"column:id;primaryKey;type:varchar(64)"`
	EpisodeID       string                `json:"episode_id" gorm:"column:episode_id;type:varchar(64);not null;index:idx_medical_authorization_episode_id"`
	FamilyID        string                `json:"family_id" gorm:"column:family_id;type:varchar(64);not null"`
	PetID           string                `json:"pet_id" gorm:"column:pet_id;type:varchar(64);not null"`
	AuthorizedBy    string                `json:"authorized_by" gorm:"column:authorized_by;type:varchar(64);not null"`
	RecipientType   string                `json:"recipient_type" gorm:"column:recipient_type;type:varchar(32);not null"`
	RecipientID     string                `json:"recipient_id" gorm:"column:recipient_id;type:varchar(64)"`
	DataScopesJSON  string                `json:"data_scopes_json" gorm:"column:data_scopes_json;type:jsonb"`
	ExpiresAt       types.NullableCSTTime `json:"expires_at" gorm:"column:expires_at;type:timestamptz"`
	Status          string                `json:"status" gorm:"column:status;type:varchar(32);not null;index:idx_medical_authorization_status"`
	CreatedBy       string                `json:"created_by" gorm:"column:created_by;type:varchar(64)"`
	CreatedAt       types.CSTTime         `json:"created_at" gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt       types.CSTTime         `json:"updated_at" gorm:"column:updated_at;type:timestamptz;not null"`
	RevokedAt       types.NullableCSTTime `json:"revoked_at" gorm:"column:revoked_at;type:timestamptz"`
	DeletedAt       gorm.DeletedAt        `json:"-" gorm:"column:deleted_at;index:idx_medical_authorization_deleted_at"`
}

func (MedicalAuthorization) TableName() string { return "medical_authorization" }

// 联合索引：(family_id, pet_id) 按家庭+宠物查询授权
func (MedicalAuthorization) Indexes() map[string][]string {
	return map[string][]string{
		"idx_medical_authorization_family_pet": {"family_id", "pet_id"},
	}
}
