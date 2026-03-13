package memory

// DefaultTokenBudget is the default total token budget for memory injection.
const DefaultTokenBudget = 2000

// EstTokensPerChar approximates tokens from character count (~4 chars per token).
const EstTokensPerChar = 4

// EstimateTokens returns an approximate token count for text.
func EstimateTokens(s string) int {
	n := len(s) / EstTokensPerChar
	if n < 1 && len(s) > 0 {
		return 1
	}
	return n
}

// BudgetAllocator allocates a token budget across tiers in priority order.
// Unused budget from one tier rolls to the next.
// Order: working > episodic > semantic > procedural > relationships.
type BudgetAllocator struct {
	TotalTokens int
}

// NewBudgetAllocator creates an allocator with the given total token budget.
func NewBudgetAllocator(totalTokens int) *BudgetAllocator {
	if totalTokens <= 0 {
		totalTokens = DefaultTokenBudget
	}
	return &BudgetAllocator{TotalTokens: totalTokens}
}

// Allocate trims content slices to fit within the budget, in priority order.
// Returns (working, episodic, facts, goals, procedures, relationships) counts of items that fit.
// Each content entry is a string; we estimate its tokens and stop when budget exceeded.
func (b *BudgetAllocator) Allocate(
	working, episodic []string,
	facts, goals []string,
	procedures []ProcedureEntry,
	relationships []RelationshipEntry,
) (
	workingOut, episodicOut []string,
	factsOut, goalsOut []string,
	proceduresOut []ProcedureEntry,
	relationshipsOut []RelationshipEntry,
) {
	remaining := b.TotalTokens

	workingOut = fitToBudget(remaining, working, func(s string) int { return EstimateTokens(s) })
	for _, s := range workingOut {
		remaining -= EstimateTokens(s)
	}
	if remaining <= 0 {
		return workingOut, nil, nil, nil, nil, nil
	}

	episodicOut = fitToBudget(remaining, episodic, func(s string) int { return EstimateTokens(s) })
	for _, s := range episodicOut {
		remaining -= EstimateTokens(s)
	}
	if remaining <= 0 {
		return workingOut, episodicOut, nil, nil, nil, nil
	}

	factsOut = fitToBudget(remaining, facts, func(s string) int { return EstimateTokens(s) })
	for _, s := range factsOut {
		remaining -= EstimateTokens(s)
	}
	if remaining <= 0 {
		return workingOut, episodicOut, factsOut, nil, nil, nil
	}

	goalsOut = fitToBudget(remaining, goals, func(s string) int { return EstimateTokens(s) })
	for _, s := range goalsOut {
		remaining -= EstimateTokens(s)
	}
	if remaining <= 0 {
		return workingOut, episodicOut, factsOut, goalsOut, nil, nil
	}

	proceduresOut = fitProceduresToBudget(remaining, procedures)
	for _, p := range proceduresOut {
		remaining -= EstimateTokens(p.Name) + 10 // ~10 tokens for "N steps" etc
	}
	if remaining <= 0 {
		return workingOut, episodicOut, factsOut, goalsOut, proceduresOut, nil
	}

	relationshipsOut = fitRelationshipsToBudget(remaining, relationships)
	return workingOut, episodicOut, factsOut, goalsOut, proceduresOut, relationshipsOut
}

func fitToBudget(budget int, items []string, est func(string) int) []string {
	var out []string
	for _, s := range items {
		need := est(s)
		if need > budget {
			break
		}
		out = append(out, s)
		budget -= need
	}
	return out
}

func fitProceduresToBudget(budget int, items []ProcedureEntry) []ProcedureEntry {
	var out []ProcedureEntry
	for _, p := range items {
		need := EstimateTokens(p.Name) + 10
		if need > budget {
			break
		}
		out = append(out, p)
		budget -= need
	}
	return out
}

func fitRelationshipsToBudget(budget int, items []RelationshipEntry) []RelationshipEntry {
	var out []RelationshipEntry
	for _, r := range items {
		need := EstimateTokens(r.Address) + EstimateTokens(r.Name) + 20
		if need > budget {
			break
		}
		out = append(out, r)
		budget -= need
	}
	return out
}
