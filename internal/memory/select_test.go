package memory

import (
	"context"
	"strings"
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

// mockTierStore returns fixed data for testing.
type mockTierStore struct {
	working     []state.WorkingMemoryRow
	episodic    []state.EpisodicMemoryRow
	semantic    []state.SemanticMemoryRow
	procedural  []state.ProceduralMemoryRow
	relationship []state.RelationshipMemoryRow
}

func (m *mockTierStore) GetWorkingMemory(sessionID string, limit int) ([]state.WorkingMemoryRow, error) {
	return m.working, nil
}
func (m *mockTierStore) GetEpisodicMemory(sessionID string, limit int) ([]state.EpisodicMemoryRow, error) {
	return m.episodic, nil
}
func (m *mockTierStore) GetSemanticMemory(limit int) ([]state.SemanticMemoryRow, error) {
	return m.semantic, nil
}
func (m *mockTierStore) GetProceduralMemory(limit int) ([]state.ProceduralMemoryRow, error) {
	return m.procedural, nil
}
func (m *mockTierStore) GetRelationshipMemory(limit int) ([]state.RelationshipMemoryRow, error) {
	return m.relationship, nil
}

func TestTieredMemorySelector_Select_Empty(t *testing.T) {
	store := &mockTierStore{}
	sel := NewTieredMemorySelector(store, DefaultTierConfig())
	res, err := sel.Select(context.Background(), "", "", 2000)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if res.Block != "" {
		t.Errorf("expected empty block, got %q", res.Block)
	}
	if len(res.Stats) != 0 {
		t.Errorf("expected no stats, got %v", res.Stats)
	}
}

func TestTieredMemorySelector_Select_WithData(t *testing.T) {
	store := &mockTierStore{
		working: []state.WorkingMemoryRow{
			{Content: "Current goal: deploy to prod"},
		},
		semantic: []state.SemanticMemoryRow{
			{Value: "Prod URL: https://example.com"},
		},
	}
	sel := NewTieredMemorySelector(store, DefaultTierConfig())
	res, err := sel.Select(context.Background(), "", "", 2000)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if !strings.Contains(res.Block, "## Memory") {
		t.Errorf("expected ## Memory, got %q", res.Block[:50])
	}
	if !strings.Contains(res.Block, "Working Memory") {
		t.Error("missing Working Memory section")
	}
	if !strings.Contains(res.Block, "Known Facts") {
		t.Error("missing Known Facts section")
	}
	if res.Stats["working"] != 1 {
		t.Errorf("expected working=1, got %d", res.Stats["working"])
	}
	if res.Stats["semantic"] != 1 {
		t.Errorf("expected semantic=1, got %d", res.Stats["semantic"])
	}
}

func TestTieredMemorySelector_Select_RespectsBudget(t *testing.T) {
	store := &mockTierStore{
		working: []state.WorkingMemoryRow{
			{Content: strings.Repeat("x", 400)}, // ~100 tokens
		},
		semantic: []state.SemanticMemoryRow{
			{Value: strings.Repeat("y", 800)}, // ~200 tokens
		},
	}
	sel := NewTieredMemorySelector(store, DefaultTierConfig())
	res, err := sel.Select(context.Background(), "", "", 150) // small budget
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	// Should fit working (~100 tokens) but maybe not all of semantic
	if res.Used > 200 {
		t.Errorf("used %d tokens, budget was 150", res.Used)
	}
}

func TestDefaultTierConfig(t *testing.T) {
	cfg := DefaultTierConfig()
	if cfg["working"].Soft != 1800 {
		t.Errorf("working.Soft = %d, want 1800", cfg["working"].Soft)
	}
	if cfg["episodic"].Hard != 2000 {
		t.Errorf("episodic.Hard = %d, want 2000", cfg["episodic"].Hard)
	}
}

func TestTieredMemoryRetriever_Retrieve(t *testing.T) {
	store := &mockTierStore{
		working: []state.WorkingMemoryRow{{Content: "test"}},
	}
	r := NewTieredMemoryRetriever(store, DefaultTierConfig())
	block, err := r.Retrieve(context.Background(), "", "input")
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if !strings.Contains(block, "test") {
		t.Errorf("expected 'test' in block, got %q", block)
	}
}

func TestTieredMemoryRetriever_RetrieveWithBudget(t *testing.T) {
	store := &mockTierStore{
		working: []state.WorkingMemoryRow{{Content: "goal"}},
	}
	r := NewTieredMemoryRetriever(store, DefaultTierConfig())
	block, stats, err := r.RetrieveWithBudget(context.Background(), "", "input", 500)
	if err != nil {
		t.Fatalf("RetrieveWithBudget: %v", err)
	}
	if !strings.Contains(block, "goal") {
		t.Errorf("expected 'goal' in block, got %q", block)
	}
	if stats["working"] != 1 {
		t.Errorf("expected working=1, got %v", stats)
	}
}
