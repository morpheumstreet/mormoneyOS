package memory

import (
	"context"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

// Memory5TierStore provides read access to the 5-tier memory tables.
type Memory5TierStore interface {
	GetWorkingMemory(sessionID string, limit int) ([]state.WorkingMemoryRow, error)
	GetEpisodicMemory(sessionID string, limit int) ([]state.EpisodicMemoryRow, error)
	GetSemanticMemory(limit int) ([]state.SemanticMemoryRow, error)
	GetProceduralMemory(limit int) ([]state.ProceduralMemoryRow, error)
	GetRelationshipMemory(limit int) ([]state.RelationshipMemoryRow, error)
}

// DBMemoryRetriever retrieves memories from the 5-tier DB tables (Phase 2).
// Applies priority order and token budget.
type DBMemoryRetriever struct {
	store  Memory5TierStore
	budget *BudgetAllocator
}

// NewDBMemoryRetriever creates a retriever that reads from the 5-tier tables.
func NewDBMemoryRetriever(store Memory5TierStore, budget *BudgetAllocator) *DBMemoryRetriever {
	if budget == nil {
		budget = NewBudgetAllocator(DefaultTokenBudget)
	}
	return &DBMemoryRetriever{store: store, budget: budget}
}

// RetrieveBlock fetches from all five tables, applies budget, and returns MemoryBlock.
func (r *DBMemoryRetriever) RetrieveBlock(ctx context.Context, sessionID string, currentInput string) (*MemoryBlock, error) {
	_ = ctx
	_ = currentInput

	// Fetch all tiers (generous limits; budget will trim)
	working, _ := r.store.GetWorkingMemory(sessionID, 50)
	episodic, _ := r.store.GetEpisodicMemory(sessionID, 50)
	semantic, _ := r.store.GetSemanticMemory(100)
	procedural, _ := r.store.GetProceduralMemory(50)
	relationships, _ := r.store.GetRelationshipMemory(50)

	// Convert to MemoryBlock format
	workingStr := make([]string, 0, len(working))
	for _, w := range working {
		workingStr = append(workingStr, w.Content)
	}

	episodicStr := make([]string, 0, len(episodic))
	for _, e := range episodic {
		s := e.Summary
		if e.Outcome != "" {
			s += " [" + e.Outcome + "]"
		}
		episodicStr = append(episodicStr, s)
	}

	facts := make([]string, 0, len(semantic))
	for _, s := range semantic {
		facts = append(facts, s.Value)
	}

	procEntries := make([]ProcedureEntry, 0, len(procedural))
	for _, p := range procedural {
		steps := countStepsFromString(p.Steps)
		procEntries = append(procEntries, ProcedureEntry{Name: p.Name, Steps: steps})
	}

	relEntries := make([]RelationshipEntry, 0, len(relationships))
	for _, rel := range relationships {
		relEntries = append(relEntries, RelationshipEntry{
			Address:    rel.EntityAddress,
			Name:       rel.EntityName,
			Type:       rel.RelationshipType,
			TrustScore: rel.TrustScore,
			Count:      rel.InteractionCount,
		})
	}

	// Allocate budget (priority: working > episodic > semantic > procedural > relationships)
	// Goals from KV are not in 5-tier tables; DB retriever omits them unless we add a fallback
	workingOut, episodicOut, factsOut, _, proceduresOut, relationshipsOut := r.budget.Allocate(
		workingStr, episodicStr,
		facts, nil, // no goals from DB
		procEntries, relEntries,
	)

	block := &MemoryBlock{
		Working:      workingOut,
		Episodic:     episodicOut,
		Facts:        factsOut,
		Goals:        nil,
		Procedures:   proceduresOut,
		Relationships: relationshipsOut,
	}

	return block, nil
}

// Retrieve fetches from all five tables, applies budget, and returns formatted block.
func (r *DBMemoryRetriever) Retrieve(ctx context.Context, sessionID string, currentInput string) (string, error) {
	block, err := r.RetrieveBlock(ctx, sessionID, currentInput)
	if err != nil {
		return "", err
	}
	return FormatMemoryBlock(block), nil
}

func countStepsFromString(steps string) int {
	n := 0
	for _, line := range strings.Split(steps, "\n") {
		if strings.TrimSpace(line) != "" {
			n++
		}
	}
	if n == 0 && strings.TrimSpace(steps) != "" {
		return 1
	}
	return n
}
