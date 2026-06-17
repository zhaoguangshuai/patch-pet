package memory

import (
	"testing"
	"time"
)

func TestSessionMemoryStruct(t *testing.T) {
	mem := SessionMemory{
		SessionID: "sess_123",
		Context:   map[string]any{"key": "value"},
		ExpireAt:  time.Now().Add(SessionTTL).Unix(),
	}
	if mem.SessionID != "sess_123" {
		t.Errorf("SessionID = %s, want sess_123", mem.SessionID)
	}
	if mem.Context["key"] != "value" {
		t.Error("Context should contain key=value")
	}
}

func TestEpisodeMemoryStruct(t *testing.T) {
	mem := EpisodeMemory{
		AggregateID: "ep_123",
		Data:        map[string]any{"status": "active"},
	}
	if mem.AggregateID != "ep_123" {
		t.Errorf("AggregateID = %s, want ep_123", mem.AggregateID)
	}
}

func TestFamilyMemoryStruct(t *testing.T) {
	mem := FamilyMemory{
		FamilyID: "fam_123",
		Config:   map[string]any{"pets": []string{"pet1", "pet2"}},
	}
	if mem.FamilyID != "fam_123" {
		t.Errorf("FamilyID = %s, want fam_123", mem.FamilyID)
	}
}

func TestSessionTTLConstant(t *testing.T) {
	if SessionTTL != 24*time.Hour {
		t.Errorf("SessionTTL = %v, want 24h", SessionTTL)
	}
}

func TestEpisodeRetentionDays(t *testing.T) {
	if EpisodeRetentionDays != 180 {
		t.Errorf("EpisodeRetentionDays = %d, want 180", EpisodeRetentionDays)
	}
}

func TestFamilyCacheTTL(t *testing.T) {
	if FamilyCacheTTL != 1*time.Hour {
		t.Errorf("FamilyCacheTTL = %v, want 1h", FamilyCacheTTL)
	}
}

func TestEpisodeMemoryEntityTableName(t *testing.T) {
	e := EpisodeMemoryEntity{}
	if e.TableName() != "episode_memory" {
		t.Errorf("TableName = %s, want episode_memory", e.TableName())
	}
}

func TestFamilyMemoryEntityTableName(t *testing.T) {
	e := FamilyMemoryEntity{}
	if e.TableName() != "family_memory" {
		t.Errorf("TableName = %s, want family_memory", e.TableName())
	}
}

func TestManagerInterface(t *testing.T) {
	// Verify MemoryManager implements Manager interface
	var _ Manager = (*MemoryManager)(nil)
}
