package memory

import (
	"context"
	"fmt"
	"strings"
)

// TierLimits defines per-tier soft and hard token budgets.
type TierLimits struct {
	Soft int `json:"soft"` // Preferred cap; when over, drop oldest/lowest priority
	Hard int `json:"hard"` // Absolute max for this tier
}

// TierConfig holds limits for all memory tiers.
// Keys: "working", "episodic", "semantic", "procedural", "relationship"
type TierConfig map[string]TierLimits

// DefaultTierConfig returns sensible per-tier limits (design-aligned).
func DefaultTierConfig() TierConfig {
	return TierConfig{
		"working":      {Soft: 1800, Hard: 2200},
		"episodic":     {Soft: 1500, Hard: 2000},
		"semantic":     {Soft: 1200, Hard: 1500},
		"procedural":   {Soft: 800, Hard: 1000},
		"relationship": {Soft: 600, Hard: 800},
	}
}

// TieredMemorySelector selects memory entries respecting per-tier budgets.
// Priority order: working > episodic > semantic > procedural > relationship.
type TieredMemorySelector struct {
	store  Memory5TierStore
	config TierConfig
}

// NewTieredMemorySelector creates a selector with the given store and config.
func NewTieredMemorySelector(store Memory5TierStore, config TierConfig) *TieredMemorySelector {
	if config == nil {
		config = DefaultTierConfig()
	}
	return &TieredMemorySelector{store: store, config: config}
}

// SelectResult holds the formatted block, token estimate, and per-tier stats.
type SelectResult struct {
	Block string
	Used  int
	Stats map[string]int // tier -> count of items included
}

// Select fetches from all tiers, applies per-tier soft caps, and returns formatted block.
// totalBudget is the remaining context budget for memory; each tier gets min(limits.Soft, remaining).
func (s *TieredMemorySelector) Select(ctx context.Context, sessionID, input string, totalBudget int) (SelectResult, error) {
	_ = ctx
	_ = input

	stats := make(map[string]int)
	remaining := totalBudget
	if remaining <= 0 {
		remaining = DefaultTokenBudget
	}

	// Priority order
	tiers := []string{"working", "episodic", "semantic", "procedural", "relationship"}

	var sections []string

	for _, tierName := range tiers {
		limits, ok := s.config[tierName]
		if !ok || limits.Soft <= 0 {
			continue
		}
		tierBudget := limits.Soft
		if tierBudget > remaining {
			tierBudget = remaining
		}
		if tierBudget <= 0 {
			break
		}

		var content string
		var used int
		var count int

		switch tierName {
		case "working":
			content, used, count = s.selectWorking(sessionID, tierBudget)
		case "episodic":
			content, used, count = s.selectEpisodic(sessionID, tierBudget)
		case "semantic":
			content, used, count = s.selectSemantic(tierBudget)
		case "procedural":
			content, used, count = s.selectProcedural(tierBudget)
		case "relationship":
			content, used, count = s.selectRelationship(tierBudget)
		}

		if content != "" {
			sections = append(sections, content)
			remaining -= used
			stats[tierName] = count
		}
	}

	block := ""
	if len(sections) > 0 {
		block = "## Memory\n\n" + strings.Join(sections, "\n\n")
	}

	totalUsed := totalBudget - remaining
	return SelectResult{Block: block, Used: totalUsed, Stats: stats}, nil
}

func (s *TieredMemorySelector) selectWorking(sessionID string, budget int) (string, int, int) {
	rows, _ := s.store.GetWorkingMemory(sessionID, 50)
	items := make([]string, 0, len(rows))
	for _, r := range rows {
		items = append(items, r.Content)
	}
	out := fitToBudget(budget, items, EstimateTokens)
	var used int
	for _, x := range out {
		used += EstimateTokens(x)
	}
	if len(out) == 0 {
		return "", 0, 0
	}
	lines := make([]string, len(out))
	for i, x := range out {
		lines[i] = "- " + x
	}
	return "### Working Memory\n" + strings.Join(lines, "\n"), used, len(out)
}

func (s *TieredMemorySelector) selectEpisodic(sessionID string, budget int) (string, int, int) {
	rows, _ := s.store.GetEpisodicMemory(sessionID, 50)
	items := make([]string, 0, len(rows))
	for _, e := range rows {
		s := e.Summary
		if e.Outcome != "" {
			s += " [" + e.Outcome + "]"
		}
		items = append(items, s)
	}
	out := fitToBudget(budget, items, EstimateTokens)
	var used int
	for _, x := range out {
		used += EstimateTokens(x)
	}
	if len(out) == 0 {
		return "", 0, 0
	}
	lines := make([]string, len(out))
	for i, x := range out {
		lines[i] = "- " + x
	}
	return "### Episodic Memory\n" + strings.Join(lines, "\n"), used, len(out)
}

func (s *TieredMemorySelector) selectSemantic(budget int) (string, int, int) {
	rows, _ := s.store.GetSemanticMemory(100)
	items := make([]string, 0, len(rows))
	for _, r := range rows {
		items = append(items, r.Value)
	}
	out := fitToBudget(budget, items, EstimateTokens)
	var used int
	for _, x := range out {
		used += EstimateTokens(x)
	}
	if len(out) == 0 {
		return "", 0, 0
	}
	lines := make([]string, len(out))
	for i, x := range out {
		lines[i] = "- " + x
	}
	return "### Known Facts\n" + strings.Join(lines, "\n"), used, len(out)
}

func (s *TieredMemorySelector) selectProcedural(budget int) (string, int, int) {
	rows, _ := s.store.GetProceduralMemory(50)
	items := make([]ProcedureEntry, 0, len(rows))
	for _, p := range rows {
		items = append(items, ProcedureEntry{Name: p.Name, Steps: countStepsFromString(p.Steps)})
	}
	out := fitProceduresToBudget(budget, items)
	var used int
	for _, p := range out {
		used += EstimateTokens(p.Name) + 10
	}
	if len(out) == 0 {
		return "", 0, 0
	}
	lines := make([]string, len(out))
	for i, p := range out {
		lines[i] = "- " + p.Name + ": " + fmt.Sprintf("%d", p.Steps) + " steps"
	}
	return "### Known Procedures\n" + strings.Join(lines, "\n"), used, len(out)
}

func (s *TieredMemorySelector) selectRelationship(budget int) (string, int, int) {
	rows, _ := s.store.GetRelationshipMemory(50)
	items := make([]RelationshipEntry, 0, len(rows))
	for _, r := range rows {
		items = append(items, RelationshipEntry{
			Address: r.EntityAddress,
			Name:    r.EntityName,
			Type:    r.RelationshipType,
			TrustScore: r.TrustScore,
			Count:   r.InteractionCount,
		})
	}
	out := fitRelationshipsToBudget(budget, items)
	var used int
	for _, r := range out {
		used += EstimateTokens(r.Address) + EstimateTokens(r.Name) + 20
	}
	if len(out) == 0 {
		return "", 0, 0
	}
	lines := make([]string, len(out))
	for i, rel := range out {
		s := rel.Address
		if rel.Name != "" {
			s = rel.Name + " (" + rel.Address + ")"
		}
		if rel.Type != "" {
			s += " [" + rel.Type + "]"
		}
		s += fmt.Sprintf(" trust=%.2f interactions=%d", rel.TrustScore, rel.Count)
		lines[i] = "- " + s
	}
	return "### Relationships\n" + strings.Join(lines, "\n"), used, len(out)
}
