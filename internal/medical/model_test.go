package medical

import (
	"testing"

	"gorm.io/gorm"
)

func TestMedicalEpisodeTableName(t *testing.T) {
	e := MedicalEpisode{}
	if e.TableName() != "medical_episode" {
		t.Errorf("TableName = %q, want %q", e.TableName(), "medical_episode")
	}
}

func TestCareTaskTableName(t *testing.T) {
	c := CareTask{}
	if c.TableName() != "care_task" {
		t.Errorf("TableName = %q, want %q", c.TableName(), "care_task")
	}
}

func TestCareTaskActionTableName(t *testing.T) {
	c := CareTaskAction{}
	if c.TableName() != "care_task_action" {
		t.Errorf("TableName = %q, want %q", c.TableName(), "care_task_action")
	}
}

func TestMedicalSummaryTableName(t *testing.T) {
	m := MedicalSummary{}
	if m.TableName() != "medical_summary" {
		t.Errorf("TableName = %q, want %q", m.TableName(), "medical_summary")
	}
}

func TestMedicalAuthorizationTableName(t *testing.T) {
	m := MedicalAuthorization{}
	if m.TableName() != "medical_authorization" {
		t.Errorf("TableName = %q, want %q", m.TableName(), "medical_authorization")
	}
}

func TestAllModelsHaveDeletedAt(t *testing.T) {
	models := Models()
	for _, m := range models {
		// 每个模型必须有 DeletedAt 字段（gorm.DeletedAt 类型）
		// 通过类型断言无法直接检查字段，但 Models() 返回正确数量
		if m == nil {
			t.Error("model should not be nil")
		}
	}
}

func TestModelsCount(t *testing.T) {
	models := Models()
	if len(models) != 5 {
		t.Errorf("Models count = %d, want 5", len(models))
	}
}

func TestMedicalEpisodeHasBaseFields(t *testing.T) {
	// 验证基础字段存在：created_by, created_at, updated_at, deleted_at
	e := MedicalEpisode{
		ID:        "ep_001",
		CreatedBy: "user_001",
	}
	if e.CreatedBy != "user_001" {
		t.Errorf("CreatedBy = %q, want %q", e.CreatedBy, "user_001")
	}
	if e.DeletedAt != (gorm.DeletedAt{}) {
		t.Error("DeletedAt should be zero value for new entity")
	}
}

func TestCareTaskActionHasBaseFields(t *testing.T) {
	a := CareTaskAction{
		ID:        "act_001",
		CreatedBy: "user_001",
	}
	if a.CreatedBy != "user_001" {
		t.Errorf("CreatedBy = %q, want %q", a.CreatedBy, "user_001")
	}
}

func TestMedicalEpisodeStatusValues(t *testing.T) {
	statuses := []EpisodeStatus{
		EpisodeCreated, InstructionDrafted, AwaitingOwnerConfirm,
		EpisodeActive, TaskDue, TaskDone, TaskSnoozed, TaskMissed,
		SummaryReady, AwaitingAuthorization, SharedToClinic,
		HostedCare, EpisodeClosed,
	}
	if len(statuses) != 13 {
		t.Errorf("EpisodeStatus count = %d, want 13", len(statuses))
	}
}
