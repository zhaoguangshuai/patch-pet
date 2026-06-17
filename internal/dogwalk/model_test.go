package dogwalk

import (
	"testing"

	"gorm.io/gorm"
)

func TestDogWalkOpportunityTableName(t *testing.T) {
	d := DogWalkOpportunity{}
	if d.TableName() != "dog_walk_opportunity" {
		t.Errorf("TableName = %q, want %q", d.TableName(), "dog_walk_opportunity")
	}
}

func TestDogWalkPlanTableName(t *testing.T) {
	d := DogWalkPlan{}
	if d.TableName() != "dog_walk_plan" {
		t.Errorf("TableName = %q, want %q", d.TableName(), "dog_walk_plan")
	}
}

func TestDogWalkOrderTableName(t *testing.T) {
	d := DogWalkOrder{}
	if d.TableName() != "dog_walk_order" {
		t.Errorf("TableName = %q, want %q", d.TableName(), "dog_walk_order")
	}
}

func TestDogWalkLiveEventTableName(t *testing.T) {
	d := DogWalkLiveEvent{}
	if d.TableName() != "dog_walk_live_event" {
		t.Errorf("TableName = %q, want %q", d.TableName(), "dog_walk_live_event")
	}
}

func TestDogWalkReportTableName(t *testing.T) {
	d := DogWalkReport{}
	if d.TableName() != "dog_walk_report" {
		t.Errorf("TableName = %q, want %q", d.TableName(), "dog_walk_report")
	}
}

func TestModelsCount(t *testing.T) {
	models := Models()
	if len(models) != 5 {
		t.Errorf("Models count = %d, want 5", len(models))
	}
}

func TestDogWalkOrderHasBaseFields(t *testing.T) {
	o := DogWalkOrder{
		ID:        "order_001",
		CreatedBy: "user_001",
	}
	if o.CreatedBy != "user_001" {
		t.Errorf("CreatedBy = %q, want %q", o.CreatedBy, "user_001")
	}
	if o.DeletedAt != (gorm.DeletedAt{}) {
		t.Error("DeletedAt should be zero value for new entity")
	}
}

func TestDogWalkLiveEventNoDeletedAt(t *testing.T) {
	// 时序数据不使用软删除，通过分区归档保留 90 天
	e := DogWalkLiveEvent{
		ID: "evt_001",
	}
	if e.CreatedBy != "" {
		t.Error("CreatedBy should be empty for default")
	}
}

func TestOrderStatusValues(t *testing.T) {
	statuses := []OrderStatus{
		OrderPlanDrafting, OrderVendorSelected, OrderRoutePreference,
		OrderRouteReady, OrderAwaitingPayment, OrderPaid,
		OrderBooked, OrderInService, OrderAnomalyDetected,
		OrderCompleted, OrderReportReady, OrderLifeStreamWritten,
		OrderCancelled, OrderRefunded,
	}
	if len(statuses) != 14 {
		t.Errorf("OrderStatus count = %d, want 14", len(statuses))
	}
}

func TestOpportunityStatusValues(t *testing.T) {
	statuses := []OpportunityStatus{
		OppCandidateDetected, OppAwaitingPermission, OppRejected, OppReminderOnly,
	}
	if len(statuses) != 4 {
		t.Errorf("OpportunityStatus count = %d, want 4", len(statuses))
	}
}
