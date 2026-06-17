package database

import (
	"testing"
)

func TestWithCondition(t *testing.T) {
	cfg := &QueryConfig{}
	WithCondition("status = ?", "active")(cfg)

	if len(cfg.Conditions) != 1 {
		t.Fatalf("Conditions len = %d, want 1", len(cfg.Conditions))
	}
	if cfg.Conditions[0].Query != "status = ?" {
		t.Errorf("Query = %q, want %q", cfg.Conditions[0].Query, "status = ?")
	}
	if len(cfg.Conditions[0].Args) != 1 || cfg.Conditions[0].Args[0] != "active" {
		t.Errorf("Args = %v, want [active]", cfg.Conditions[0].Args)
	}
}

func TestWithMultipleConditions(t *testing.T) {
	cfg := &QueryConfig{}
	WithCondition("status = ?", "active")(cfg)
	WithCondition("family_id = ?", "fam_001")(cfg)

	if len(cfg.Conditions) != 2 {
		t.Fatalf("Conditions len = %d, want 2", len(cfg.Conditions))
	}
	if cfg.Conditions[0].Query != "status = ?" {
		t.Errorf("first condition Query = %q", cfg.Conditions[0].Query)
	}
	if cfg.Conditions[1].Query != "family_id = ?" {
		t.Errorf("second condition Query = %q", cfg.Conditions[1].Query)
	}
}

func TestWithOrderBy(t *testing.T) {
	cfg := &QueryConfig{}
	WithOrderBy("created_at DESC")(cfg)

	if cfg.OrderBy != "created_at DESC" {
		t.Errorf("OrderBy = %q, want %q", cfg.OrderBy, "created_at DESC")
	}
}

func TestWithPagination(t *testing.T) {
	cfg := &QueryConfig{}
	WithPagination(3, 20)(cfg)

	if cfg.PageNum != 3 {
		t.Errorf("PageNum = %d, want 3", cfg.PageNum)
	}
	if cfg.PageSize != 20 {
		t.Errorf("PageSize = %d, want 20", cfg.PageSize)
	}
}

func TestWithPreload(t *testing.T) {
	cfg := &QueryConfig{}
	WithPreload("Tasks")(cfg)
	WithPreload("Authorizations")(cfg)

	if len(cfg.Preloads) != 2 {
		t.Fatalf("Preloads len = %d, want 2", len(cfg.Preloads))
	}
	if cfg.Preloads[0] != "Tasks" {
		t.Errorf("Preloads[0] = %q, want %q", cfg.Preloads[0], "Tasks")
	}
	if cfg.Preloads[1] != "Authorizations" {
		t.Errorf("Preloads[1] = %q, want %q", cfg.Preloads[1], "Authorizations")
	}
}

func TestQueryOptionDefaults(t *testing.T) {
	cfg := &QueryConfig{
		PageNum:  1,
		PageSize: 20,
	}

	if cfg.PageNum != 1 {
		t.Errorf("default PageNum = %d, want 1", cfg.PageNum)
	}
	if cfg.PageSize != 20 {
		t.Errorf("default PageSize = %d, want 20", cfg.PageSize)
	}
	if len(cfg.Conditions) != 0 {
		t.Errorf("default Conditions len = %d, want 0", len(cfg.Conditions))
	}
	if cfg.OrderBy != "" {
		t.Errorf("default OrderBy = %q, want empty", cfg.OrderBy)
	}
}

func TestMultipleOptionsApplied(t *testing.T) {
	cfg := &QueryConfig{
		PageNum:  1,
		PageSize: 20,
	}

	opts := []QueryOption{
		WithCondition("pet_id = ?", "pet_001"),
		WithCondition("status = ?", "Active"),
		WithOrderBy("created_at DESC"),
		WithPagination(2, 10),
		WithPreload("Tasks"),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if len(cfg.Conditions) != 2 {
		t.Errorf("Conditions len = %d, want 2", len(cfg.Conditions))
	}
	if cfg.OrderBy != "created_at DESC" {
		t.Errorf("OrderBy = %q", cfg.OrderBy)
	}
	if cfg.PageNum != 2 {
		t.Errorf("PageNum = %d, want 2", cfg.PageNum)
	}
	if cfg.PageSize != 10 {
		t.Errorf("PageSize = %d, want 10", cfg.PageSize)
	}
	if len(cfg.Preloads) != 1 {
		t.Errorf("Preloads len = %d, want 1", len(cfg.Preloads))
	}
}
