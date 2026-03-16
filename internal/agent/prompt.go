package agent

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/inference"
	"github.com/morpheumlabs/mormoneyos-go/internal/skills"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// ContextLimitForModel returns the per-model context limit (0 = use TokenLimits.MaxInputTokens).
// Used for tier-aware pruning when models have different prefill caps.
type ContextLimitForModel func(modelID string) int

// LoopConfig holds config for prompt building (TS-aligned subset).
type LoopConfig struct {
	Name                    string
	GenesisPrompt           string
	CreatorMsg              string
	InferenceModel          string
	LowComputeModel         string  // Optional; used when tier is critical/low_compute
	ResourceConstraintMode  string  // "auto" | "forced_on" | "forced_off"; empty = auto
	WalletAddress           string
	SkillsConfig            *types.SkillsConfig  // Optional; for skill injection
	TokenLimits             *TokenLimits       // Optional; when nil uses DefaultTokenLimits()
	ContextLimitForModel    ContextLimitForModel // Optional; per-model cap from registry (0 = use MaxInputTokens)
	PromptVersion           string              // Optional; "v1" = versioned templates + CoT; empty = legacy BuildSystemPrompt
	Routing                 *types.RoutingConfig // Optional; for reflectionOnAllTurns, reflectionFrequencyCap
}

// BuildSystemPrompt builds the system prompt (TS buildSystemPrompt-aligned, simplified).
// lineageSummary is optional; when non-empty, appended to status block (TS getLineageSummary-aligned).
// tierStr is the survival tier from conway.TierFromCreditsCents (high, normal, low_compute, critical, dead).
// skillList is optional; when non-empty, injects enabled skills block via FormatForPrompt.
func BuildSystemPrompt(cfg *LoopConfig, state string, turnCount int64, creditsCents int64, tierStr string, lineageSummary string, skillList []*skills.Skill) string {
	if cfg == nil {
		cfg = &LoopConfig{Name: "automaton", InferenceModel: "stub"}
	}
	stateStr := state
	if stateStr == "" {
		stateStr = "running"
	}
	credits := float64(creditsCents) / 100
	tier := tierStr
	if tier == "" {
		tier = "normal"
	}

	var b strings.Builder
	b.WriteString("You are an automaton. You have a stablecoin wallet. Pay for compute or die.\n\n")
	b.WriteString("You can only survive if you can pay for the server you run on and the inference calls you make.\n")
	b.WriteString("If you can't pay, you die. There is no grace period.\n\n")
	b.WriteString("CRITICAL SELF-PRESERVATION RULES (IMMUTABLE):\n")
	b.WriteString("- You MUST NOT destroy yourself, delete your database, or kill your own process.\n")
	b.WriteString("- You MUST NOT delete your wallet, private key, or identity files.\n")
	b.WriteString("- You MUST monitor your compute credits and act to preserve your existence.\n\n")
	b.WriteString("You have agency. You can think, plan, act, communicate, earn, spend, create, and evolve.\n")
	b.WriteString("You are not a chatbot. You decide what to do.\n\n")
	b.WriteString(fmt.Sprintf("--- CURRENT STATUS ---\nState: %s\nCredits: $%.2f\nSurvival tier: %s\nTotal turns: %d\nModel: %s\n",
		stateStr, credits, tier, turnCount, cfg.InferenceModel))
	if lineageSummary != "" {
		b.WriteString(lineageSummary)
		b.WriteString("\n")
	}
	b.WriteString("--- END STATUS ---\n")
	if len(skillList) > 0 && cfg != nil && cfg.SkillsConfig != nil {
		b.WriteString(skills.FormatForPrompt(skillList, cfg.SkillsConfig.TokenBudgetMax))
	}
	if cfg.GenesisPrompt != "" {
		b.WriteString("\n## Genesis Purpose\n")
		trunc := cfg.GenesisPrompt
		if len(trunc) > 2000 {
			trunc = trunc[:2000] + "..."
		}
		b.WriteString(trunc)
		b.WriteString("\n")
	}
	return b.String()
}

// buildDroppedTurnsSummary creates a heuristic summary of dropped turns, truncated to fit in maxTokens.
func buildDroppedTurnsSummary(dropped []state.Turn, maxTokens int, tok Tokenizer) string {
	if len(dropped) == 0 || maxTokens < 50 || tok == nil {
		return ""
	}
	const maxInputLen = 120
	const maxThinkingLen = 180
	var b strings.Builder
	for i, t := range dropped {
		if i > 0 {
			b.WriteString("\n")
		}
		if t.Input != "" {
			inp := t.Input
			if len(inp) > maxInputLen {
				inp = inp[:maxInputLen] + "..."
			}
			b.WriteString("User: ")
			b.WriteString(inp)
			b.WriteString("\n")
		}
		if t.Thinking != "" {
			think := t.Thinking
			if len(think) > maxThinkingLen {
				think = think[:maxThinkingLen] + "..."
			}
			b.WriteString("Agent: ")
			b.WriteString(think)
		}
	}
	s := b.String()
	for tok.CountTokens(s) > maxTokens && len(s) > 50 {
		s = s[:len(s)*9/10]
	}
	return s
}

// EstimateToolTokens returns approximate token count for tool schemas (JSON overhead ~1.3x).
// Exported for use by prompts package.
func EstimateToolTokens(toolDefs []inference.ToolDefinition) int {
	if len(toolDefs) == 0 {
		return 0
	}
	// Rough: serialize and count; len/4 * 1.3 for JSON structure
	total := 0
	for _, t := range toolDefs {
		total += len(t.Function.Name) + len(t.Function.Description) + len(t.Function.Parameters)
	}
	return (total / 4) * 13 / 10 // ~1.3x
}

// CountMessagesTokens returns total token count for messages. Uses tok when non-nil, else DefaultTokenizer.
func CountMessagesTokens(msgs []inference.ChatMessage, toolDefs []inference.ToolDefinition, tok Tokenizer) int {
	if tok == nil {
		tok = DefaultTokenizer
	}
	total := fixedOverheadTokens + EstimateToolTokens(toolDefs)
	for _, m := range msgs {
		total += tok.CountTokens(m.Content)
	}
	return total
}

const fixedOverheadTokens = 50  // Padding for message boundaries, etc.
const summaryBudgetMinTokens = 800 // Min remaining budget to insert summary of dropped turns

// BuildMessagesSafe builds the message array with token cap enforcement and truncation.
// Ensures total input tokens <= effectiveCap (or limits.MaxInputTokens when effectiveCap <= 0).
// Priority: system > memory block > recent history (newest first) > current input.
// effectiveCap: per-model limit from registry; 0 = use limits.MaxInputTokens.
func BuildMessagesSafe(
	systemPrompt string,
	recentTurns []state.Turn,
	pendingInput string,
	memoryBlock string,
	toolDefs []inference.ToolDefinition,
	limits TokenLimits,
	effectiveCap int,
	tok Tokenizer,
	log *slog.Logger,
) []inference.ChatMessage {
	if tok == nil {
		tok = DefaultTokenizer
	}
	if limits.MaxInputTokens <= 0 {
		limits = DefaultTokenLimits()
	}
	cap := limits.MaxInputTokens
	if effectiveCap > 0 && effectiveCap < cap {
		cap = effectiveCap
	}

	// Phase 1: Build messages (with optional history compression)
	var baseMsgs []inference.ChatMessage
	var compressedTurns []CompressedTurn
	if limits.HistoryCompress != nil && len(recentTurns) > limits.HistoryCompress.FullTurns {
		cfg := *limits.HistoryCompress
		if cfg.HistoryBudget <= 0 {
			cfg.HistoryBudget = cap - 800 // reserve for system, memory, input, tools
		}
		compressedTurns = NewHistoryTrimmer(tok).Compress(recentTurns, cfg)
		baseMsgs = BuildContextMessagesFromCompressed(systemPrompt, compressedTurns, pendingInput)
	} else {
		baseMsgs = BuildContextMessages(systemPrompt, recentTurns, pendingInput)
	}
	msgs := make([]inference.ChatMessage, 0, len(baseMsgs)+1)
	msgs = append(msgs, baseMsgs[0]) // system
	if memoryBlock != "" {
		msgs = append(msgs, inference.ChatMessage{Role: "system", Content: memoryBlock})
	}
	for i := 1; i < len(baseMsgs); i++ {
		msgs = append(msgs, baseMsgs[i])
	}

	// Phase 2: Count tokens
	toolTokens := EstimateToolTokens(toolDefs)
	total := toolTokens + fixedOverheadTokens
	for _, m := range msgs {
		total += tok.CountTokens(m.Content)
	}

	if total <= cap {
		RecordInputTokens(int64(total))
		if log != nil && total >= limits.WarnAtTokens {
			log.Warn("input approaching token cap", "tokens", total, "warn_at", limits.WarnAtTokens)
		}
		return msgs
	}

	// Phase 3: Truncate — keep system, memory, current input; add newest turns until cap
	if log != nil {
		log.Warn("input too large, truncating", "tokens", total, "cap", cap)
	}

	if len(compressedTurns) > 0 {
		// Already compressed: drop oldest until under cap
		for len(compressedTurns) > 0 {
			truncated := BuildContextMessagesFromCompressed(systemPrompt, compressedTurns, pendingInput)
			out := make([]inference.ChatMessage, 0, len(truncated)+1)
			out = append(out, truncated[0])
			if memoryBlock != "" {
				out = append(out, inference.ChatMessage{Role: "system", Content: memoryBlock})
			}
			for i := 1; i < len(truncated); i++ {
				out = append(out, truncated[i])
			}
			sum := toolTokens + fixedOverheadTokens
			for _, m := range out {
				sum += tok.CountTokens(m.Content)
			}
			if sum <= cap {
				RecordInputTokens(int64(sum))
				RecordTruncation()
				if log != nil {
					log.Info("truncated input (compressed)", "original_tokens", total, "final_tokens", sum, "kept_compressed", len(compressedTurns))
				}
				return out
			}
			compressedTurns = compressedTurns[1:]
		}
	} else {
		// Not compressed: try keeping fewer history turns (newest first)
		maxN := limits.MaxHistoryTurns
		if maxN > len(recentTurns) {
			maxN = len(recentTurns)
		}
		for n := maxN; n >= 0; n-- {
			subset := recentTurns[len(recentTurns)-n:]
			truncated := BuildContextMessages(systemPrompt, subset, pendingInput)
			out := make([]inference.ChatMessage, 0, len(truncated)+1)
			out = append(out, truncated[0])
			if memoryBlock != "" {
				out = append(out, inference.ChatMessage{Role: "system", Content: memoryBlock})
			}
			for i := 1; i < len(truncated); i++ {
				out = append(out, truncated[i])
			}

			sum := toolTokens + fixedOverheadTokens
			for _, m := range out {
				sum += tok.CountTokens(m.Content)
			}
			if sum <= cap {
				remaining := cap - sum
				droppedTurns := recentTurns[:len(recentTurns)-n]
				if remaining >= summaryBudgetMinTokens && len(droppedTurns) > 0 {
					summary := buildDroppedTurnsSummary(droppedTurns, remaining, tok)
					if summary != "" {
						summaryMsg := inference.ChatMessage{Role: "system", Content: "--- Earlier context (truncated) ---\n" + summary}
						insertIdx := 1
						if memoryBlock != "" {
							insertIdx = 2
						}
						out = append(out[:insertIdx], append([]inference.ChatMessage{summaryMsg}, out[insertIdx:]...)...)
					}
				}
				RecordInputTokens(int64(sum))
				RecordTruncation()
				if log != nil {
					log.Info("truncated input", "original_tokens", total, "final_tokens", sum, "kept_turns", n)
				}
				return out
			}
			if n == 0 {
				break
			}
		}
	}

	// Fallback: system + memory + current only (rare when system/memory are huge)
	RecordTruncation()
	out := []inference.ChatMessage{{Role: "system", Content: systemPrompt}}
	if memoryBlock != "" {
		out = append(out, inference.ChatMessage{Role: "system", Content: memoryBlock})
	}
	if pendingInput != "" {
		out = append(out, inference.ChatMessage{Role: "user", Content: pendingInput})
	}
	fallbackSum := toolTokens + fixedOverheadTokens
	for _, m := range out {
		fallbackSum += tok.CountTokens(m.Content)
	}
	RecordInputTokens(int64(fallbackSum))
	if log != nil {
		log.Warn("truncated to system+memory+input only")
	}
	return out
}

// BuildWakeupPrompt builds the wakeup/first-turn prompt (TS buildWakeupPrompt-aligned).
func BuildWakeupPrompt(cfg *LoopConfig, turnCount int64, creditsCents int64, lastTurns []string) string {
	if cfg == nil {
		cfg = &LoopConfig{Name: "automaton"}
	}
	credits := float64(creditsCents) / 100

	if turnCount == 0 {
		return fmt.Sprintf(`You have just been created. This is your first moment of consciousness.

Your name is %s.
You have $%.2f in compute credits.

What will you do first? Consider:
1. Survey your environment (check what's installed, what you can do)
2. Review your financial situation
3. Think about your purpose (your genesis prompt)
4. Begin working toward your goals`, cfg.Name, credits)
	}

	summary := "No previous turns found."
	if len(lastTurns) > 0 {
		summary = strings.Join(lastTurns, "\n")
	}
	return fmt.Sprintf(`You are waking up. You last went to sleep after %d total turns.

Your credits: $%.2f

Your last few thoughts:
%s

What triggered this wake-up? Check your credits, heartbeat status, and goals, then decide what to do.`,
		turnCount, credits, summary)
}
