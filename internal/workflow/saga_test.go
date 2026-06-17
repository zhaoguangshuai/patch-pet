package workflow

import (
	"context"
	"errors"
	"testing"
)

func TestSagaStepStatusConstants(t *testing.T) {
	statuses := []SagaStepStatus{
		StepPending, StepRunning, StepCompleted, StepFailed, StepCompensated,
	}
	if len(statuses) != 5 {
		t.Errorf("SagaStepStatus count = %d, want 5", len(statuses))
	}
}

func TestSagaExecuteAllSteps(t *testing.T) {
	saga := NewSaga("test", nil, nil)

	executed := []string{}
	compensated := []string{}

	saga.AddStep(SagaStep{
		Name: "step1",
		Execute: func(ctx context.Context, state *SagaState) error {
			executed = append(executed, "step1")
			return nil
		},
		Compensate: func(ctx context.Context, state *SagaState) error {
			compensated = append(compensated, "step1")
			return nil
		},
	})

	saga.AddStep(SagaStep{
		Name: "step2",
		Execute: func(ctx context.Context, state *SagaState) error {
			executed = append(executed, "step2")
			return nil
		},
		Compensate: func(ctx context.Context, state *SagaState) error {
			compensated = append(compensated, "step2")
			return nil
		},
	})

	state := &SagaState{OrderID: "order_1"}
	err := saga.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("Saga should succeed: %v", err)
	}

	if len(executed) != 2 {
		t.Errorf("Executed steps = %d, want 2", len(executed))
	}
	if len(compensated) != 0 {
		t.Errorf("Compensated steps = %d, want 0", len(compensated))
	}
}

func TestSagaCompensateOnFailure(t *testing.T) {
	saga := NewSaga("test", nil, nil)

	executed := []string{}
	compensated := []string{}

	saga.AddStep(SagaStep{
		Name: "step1",
		Execute: func(ctx context.Context, state *SagaState) error {
			executed = append(executed, "step1")
			return nil
		},
		Compensate: func(ctx context.Context, state *SagaState) error {
			compensated = append(compensated, "step1")
			return nil
		},
	})

	saga.AddStep(SagaStep{
		Name: "step2_fail",
		Execute: func(ctx context.Context, state *SagaState) error {
			executed = append(executed, "step2_fail")
			return errors.New("step2 failed")
		},
		Compensate: func(ctx context.Context, state *SagaState) error {
			compensated = append(compensated, "step2_fail")
			return nil
		},
	})

	saga.AddStep(SagaStep{
		Name: "step3",
		Execute: func(ctx context.Context, state *SagaState) error {
			executed = append(executed, "step3")
			return nil
		},
		Compensate: func(ctx context.Context, state *SagaState) error {
			compensated = append(compensated, "step3")
			return nil
		},
	})

	state := &SagaState{OrderID: "order_1"}
	err := saga.Execute(context.Background(), state)
	if err == nil {
		t.Fatal("Saga should fail on step2")
	}

	// step1 and step2 executed, step3 not
	if len(executed) != 2 {
		t.Errorf("Executed steps = %d, want 2", len(executed))
	}

	// Only step1 compensated (step2 failed, not completed)
	if len(compensated) != 1 {
		t.Errorf("Compensated steps = %d, want 1", len(compensated))
	}
	if compensated[0] != "step1" {
		t.Errorf("Compensated step = %s, want step1", compensated[0])
	}
}

func TestSagaCompensationOrder(t *testing.T) {
	saga := NewSaga("test", nil, nil)

	compOrder := []string{}

	saga.AddStep(SagaStep{
		Name: "step1",
		Execute: func(ctx context.Context, state *SagaState) error { return nil },
		Compensate: func(ctx context.Context, state *SagaState) error {
			compOrder = append(compOrder, "step1")
			return nil
		},
	})

	saga.AddStep(SagaStep{
		Name: "step2",
		Execute: func(ctx context.Context, state *SagaState) error { return nil },
		Compensate: func(ctx context.Context, state *SagaState) error {
			compOrder = append(compOrder, "step2")
			return nil
		},
	})

	saga.AddStep(SagaStep{
		Name: "step3_fail",
		Execute: func(ctx context.Context, state *SagaState) error {
			return errors.New("fail")
		},
	})

	state := &SagaState{OrderID: "order_1"}
	_ = saga.Execute(context.Background(), state)

	// Compensation should be in reverse order: step2, step1
	if len(compOrder) != 2 {
		t.Fatalf("Compensation count = %d, want 2", len(compOrder))
	}
	if compOrder[0] != "step2" || compOrder[1] != "step1" {
		t.Errorf("Compensation order = %v, want [step2, step1]", compOrder)
	}
}

func TestSagaStateInitialization(t *testing.T) {
	saga := NewSaga("test", nil, nil)
	saga.AddStep(SagaStep{
		Name:    "step1",
		Execute: func(ctx context.Context, state *SagaState) error { return nil },
	})

	state := &SagaState{OrderID: "order_1"}
	_ = saga.Execute(context.Background(), state)

	if len(state.StepStatuses) != 1 {
		t.Errorf("StepStatuses length = %d, want 1", len(state.StepStatuses))
	}
	if state.StepStatuses[0] != StepCompleted {
		t.Errorf("Step 0 status = %s, want %s", state.StepStatuses[0], StepCompleted)
	}
}

func TestDogWalkOrderSagaStepCount(t *testing.T) {
	saga := DogWalkOrderSaga(nil, nil)
	if len(saga.steps) != 5 {
		t.Errorf("DogWalkOrderSaga steps = %d, want 5", len(saga.steps))
	}
}

func TestDogWalkOrderSagaStepNames(t *testing.T) {
	saga := DogWalkOrderSaga(nil, nil)
	expected := []string{"create_order", "payment", "book_vendor", "start_service", "complete"}
	for i, name := range expected {
		if saga.steps[i].Name != name {
			t.Errorf("Step %d name = %s, want %s", i, saga.steps[i].Name, name)
		}
	}
}
