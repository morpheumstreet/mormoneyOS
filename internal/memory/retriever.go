package memory

import (
	"context"
	"fmt"
	"strings"
)

// MemoryRetriever retrieves relevant memories for context injection (TS step 6).
// Returns a formatted block or empty string. Errors must not block the agent loop.
type MemoryRetriever interface {
	Retrieve(ctx context.Context, sessionID string, currentInput string) (block string, err error)
}

// MemoryRetrieverWithBudget extends MemoryRetriever with budget-aware retrieval.
// When the agent has limited context budget, RetrieveWithBudget selects memory within that budget.
type MemoryRetrieverWithBudget interface {
	MemoryRetriever
	RetrieveWithBudget(ctx context.Context, sessionID, currentInput string, budget int) (block string, stats map[string]int, err error)
}

// TieredMemoryRetriever uses TieredMemorySelector for budget-aware retrieval.
// Implements both MemoryRetriever and MemoryRetrieverWithBudget.
type TieredMemoryRetriever struct {
	selector *TieredMemorySelector
	defaultBudget int
}

// NewTieredMemoryRetriever creates a retriever with tiered selection.
func NewTieredMemoryRetriever(store Memory5TierStore, config TierConfig) *TieredMemoryRetriever {
	if config == nil {
		config = DefaultTierConfig()
	}
	return &TieredMemoryRetriever{
		selector:      NewTieredMemorySelector(store, config),
		defaultBudget: DefaultTokenBudget,
	}
}

// Retrieve fetches memory using the default budget.
func (r *TieredMemoryRetriever) Retrieve(ctx context.Context, sessionID, currentInput string) (string, error) {
	res, err := r.selector.Select(ctx, sessionID, currentInput, r.defaultBudget)
	if err != nil {
		return "", err
	}
	return res.Block, nil
}

// RetrieveWithBudget fetches memory within the given token budget.
func (r *TieredMemoryRetriever) RetrieveWithBudget(ctx context.Context, sessionID, currentInput string, budget int) (string, map[string]int, error) {
	if budget <= 0 {
		budget = r.defaultBudget
	}
	res, err := r.selector.Select(ctx, sessionID, currentInput, budget)
	if err != nil {
		return "", nil, err
	}
	return res.Block, res.Stats, nil
}

// MemoryBlock holds sections for formatting (TS formatMemoryBlock-aligned).
// Phase 1: Known Facts, Active Goals, Known Procedures.
// Phase 2: extended with Working, Episodic, Relationship.
type MemoryBlock struct {
	// Phase 2
	Working     []string // Session-scoped: goals, observations, plans, reflections
	Episodic    []string // Past events: summary (outcome)
	Relationships []RelationshipEntry
	// Phase 1 (semantic + procedural)
	Facts       []string // Known Facts (semantic)
	Goals       []string // Active Goals (pending only)
	Procedures  []ProcedureEntry
}

// RelationshipEntry is a relationship memory entry.
type RelationshipEntry struct {
	Address    string
	Name       string
	Type       string
	TrustScore float64
	Count      int
}

// ProcedureEntry is a procedure name and step count (Phase 1 format).
type ProcedureEntry struct {
	Name  string
	Steps int
}

// FormatMemoryBlock formats retrieval result into a markdown block for context.
// TS formatMemoryBlock-aligned. Priority order: working > episodic > semantic > procedural > relationships.
// Returns "" when empty.
func FormatMemoryBlock(r *MemoryBlock) string {
	if r == nil {
		return ""
	}
	var sections []string

	// Working memory (Phase 2)
	if len(r.Working) > 0 {
		lines := []string{"### Working Memory"}
		for _, w := range r.Working {
			lines = append(lines, "- "+w)
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}

	// Episodic memory (Phase 2)
	if len(r.Episodic) > 0 {
		lines := []string{"### Episodic Memory"}
		for _, e := range r.Episodic {
			lines = append(lines, "- "+e)
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}

	// Known Facts (semantic)
	if len(r.Facts) > 0 {
		lines := []string{"### Known Facts"}
		for _, f := range r.Facts {
			lines = append(lines, "- "+f)
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}

	// Active Goals + Known Procedures
	if len(r.Goals) > 0 {
		lines := []string{"### Active Goals"}
		for _, g := range r.Goals {
			lines = append(lines, "- "+g)
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}
	if len(r.Procedures) > 0 {
		lines := []string{"### Known Procedures"}
		for _, p := range r.Procedures {
			lines = append(lines, "- "+p.Name+": "+formatSteps(p.Steps)+" steps")
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}

	// Relationships (Phase 2)
	if len(r.Relationships) > 0 {
		lines := []string{"### Relationships"}
		for _, rel := range r.Relationships {
			s := rel.Address
			if rel.Name != "" {
				s = rel.Name + " (" + rel.Address + ")"
			}
			if rel.Type != "" {
				s += " [" + rel.Type + "]"
			}
			s += fmt.Sprintf(" trust=%.2f interactions=%d", rel.TrustScore, rel.Count)
			lines = append(lines, "- "+s)
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}

	if len(sections) == 0 {
		return ""
	}
	return "## Memory\n\n" + strings.Join(sections, "\n\n")
}

func formatSteps(n int) string {
	return fmt.Sprintf("%d", n)
}
