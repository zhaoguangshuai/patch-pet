package database

import (
	"testing"
)

func TestDefaultTimescaleConfig(t *testing.T) {
	cfg := DefaultTimescaleConfig()

	if cfg.RetentionDays != 90 {
		t.Errorf("RetentionDays = %d, want 90", cfg.RetentionDays)
	}
	if cfg.ChunkInterval != "1 day" {
		t.Errorf("ChunkInterval = %q, want %q", cfg.ChunkInterval, "1 day")
	}
}

func TestTimescaleDBConstants(t *testing.T) {
	if DefaultRetentionDays != 90 {
		t.Errorf("DefaultRetentionDays = %d, want 90", DefaultRetentionDays)
	}
	if DefaultChunkInterval != "1 day" {
		t.Errorf("DefaultChunkInterval = %q, want %q", DefaultChunkInterval, "1 day")
	}
}

func TestTimescaleDBCreation(t *testing.T) {
	// TimescaleDB 是 PG 扩展，复用同一个 *gorm.DB
	// 这里测试结构体创建（不需要实际连接）
	cfg := DefaultTimescaleConfig()
	if cfg.RetentionDays <= 0 {
		t.Error("RetentionDays should be positive")
	}
}
