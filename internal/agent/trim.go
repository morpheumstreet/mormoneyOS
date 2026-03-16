package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/inference"
	"github.com/morpheumlabs/mormoneyos-go/internal/memory"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

// CompressedTurn represents either a full turn or a one-line summary for history compression.
type CompressedTurn struct {
	Full    *state.Turn // nil when Summary is used
	Summary string     // one-line summary (when Full is nil)
}

// HistoryTrimmer provides rule-based compression of conversation turns.
type HistoryTrimmer struct {
	tokenizer Tokenizer
}

// NewHistoryTrimmer creates a trimmer with the active tokenizer.
func NewHistoryTrimmer(tok Tokenizer) *HistoryTrimmer {
	if tok == nil {
		tok = DefaultTokenizer
	}
	return &HistoryTrimmer{tokenizer: tok}
}

// HistoryTrimmerConfig holds compression parameters.
type HistoryTrimmerConfig struct {
	FullTurns      int // keep last N turns in full (default 6)
	SummarizedMax  int // max turns to summarize (turns 7–20 range, default 14)
	HistoryBudget  int // max tokens for history block (0 = no limit)
}

// DefaultHistoryTrimmerConfig returns sensible defaults.
func DefaultHistoryTrimmerConfig() HistoryTrimmerConfig {
	return HistoryTrimmerConfig{
		FullTurns:     6,
		SummarizedMax: 14,
		HistoryBudget: 2200,
	}
}

// Compress compresses history according to rules:
// - full last N turns (configurable, default 6)
// - summarized turns N+1 to N+SummarizedMax (e.g. 7–20): keep only action + result line
// - drop anything older
func (h *HistoryTrimmer) Compress(turns []state.Turn, cfg HistoryTrimmerConfig) []CompressedTurn {
	if len(turns) <= cfg.FullTurns {
		out := make([]CompressedTurn, len(turns))
		for i := range turns {
			out[i] = CompressedTurn{Full: &turns[i]}
		}
		return out
	}

	// Stage 1: keep last fullTurns turns untouched
	startFull := len(turns) - cfg.FullTurns
	fullTurns := make([]CompressedTurn, 0, cfg.FullTurns)
	for i := startFull; i < len(turns); i++ {
		fullTurns = append(fullTurns, CompressedTurn{Full: &turns[i]})
	}

	// Stage 2: summarize older turns (chronological: oldest first)
	startSummarize := startFull - cfg.SummarizedMax
	if startSummarize < 0 {
		startSummarize = 0
	}
	summaries := make([]CompressedTurn, 0, cfg.SummarizedMax)
	for i := startSummarize; i < startFull; i++ {
		summary := h.summarizeTurn(turns[i])
		if summary == "" {
			continue
		}
		summaries = append(summaries, CompressedTurn{Summary: summary})
	}

	// Order: summaries first (oldest→newest), then full turns.
	kept := make([]CompressedTurn, 0, len(summaries)+len(fullTurns))
	kept = append(kept, summaries...)
	kept = append(kept, fullTurns...)

	// Stage 3: trim if still over budget (drop oldest first)
	if cfg.HistoryBudget > 0 {
		total := h.countCompressedTokens(kept)
		for total > cfg.HistoryBudget && len(kept) > 4 {
			kept = kept[1:]
			total = h.countCompressedTokens(kept)
		}
	}

	return kept
}

func (h *HistoryTrimmer) countCompressedTokens(ct []CompressedTurn) int {
	var total int
	for _, c := range ct {
		if c.Full != nil {
			total += h.tokenizer.CountTokens(c.Full.Input)
			total += h.tokenizer.CountTokens(c.Full.Thinking)
			total += h.tokenizer.CountTokens(c.Full.ToolCalls)
		} else {
			total += h.tokenizer.CountTokens(c.Summary)
		}
	}
	return total
}

// summarizeTurn creates a short, signal-preserving line from a full turn.
// Heuristic: prefer tool call + result, or first meaningful line of thought.
func (h *HistoryTrimmer) summarizeTurn(t state.Turn) string {
	// Prefer tool results (action + outcome)
	if t.ToolCalls != "" {
		s := h.summarizeToolCalls(t.ToolCalls)
		if s != "" {
			return s
		}
	}
	// Fallback: first line of thinking
	if t.Thinking != "" {
		lines := strings.Split(t.Thinking, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if len(line) > 15 && !strings.HasPrefix(line, "Thought:") {
				return truncate(line, 90)
			}
		}
		for _, line := range lines {
			if trimmed := strings.TrimSpace(line); trimmed != "" {
				return truncate(trimmed, 90)
			}
		}
	}
	// Last resort: truncated input
	if t.Input != "" {
		return truncate("User: "+t.Input, 80)
	}
	return ""
}

func (h *HistoryTrimmer) summarizeToolCalls(toolCallsJSON string) string {
	var tcList []struct {
		Name   string `json:"name"`
		Result string `json:"result"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal([]byte(toolCallsJSON), &tcList); err != nil || len(tcList) == 0 {
		return ""
	}
	var parts []string
	for _, tc := range tcList {
		if tc.Error != "" {
			parts = append(parts, fmt.Sprintf("%s: error %s", tc.Name, truncate(tc.Error, 40)))
		} else if tc.Result != "" {
			parts = append(parts, fmt.Sprintf("%s: %s", tc.Name, truncate(tc.Result, 50)))
		} else {
			parts = append(parts, tc.Name)
		}
	}
	return strings.Join(parts, "; ")
}

func truncate(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// BuildContextMessagesFromCompressed builds messages from compressed turns (full + summaries).
// Summary turns become system messages under "--- Earlier context (compressed) ---".
func BuildContextMessagesFromCompressed(
	systemPrompt string,
	compressed []CompressedTurn,
	pendingInput string,
) []inference.ChatMessage {
	msgs := []inference.ChatMessage{
		{Role: "system", Content: systemPrompt},
	}

	var summaryLines []string
	for _, c := range compressed {
		if c.Full != nil {
			// Flush any accumulated summaries as a single system message
			if len(summaryLines) > 0 {
				msgs = append(msgs, inference.ChatMessage{
					Role:    "system",
					Content: "--- Earlier context (compressed) ---\n" + strings.Join(summaryLines, "\n"),
				})
				summaryLines = nil
			}
			t := c.Full
			if t.Input != "" {
				msgs = append(msgs, inference.ChatMessage{Role: "user", Content: t.Input})
			}
			if t.Thinking != "" || t.ToolCalls != "" {
				content := t.Thinking
				if t.ToolCalls != "" {
					content = appendToolResults(content, t.ToolCalls)
				}
				msgs = append(msgs, inference.ChatMessage{Role: "assistant", Content: content})
			}
		} else {
			summaryLines = append(summaryLines, "- "+c.Summary)
		}
	}
	if len(summaryLines) > 0 {
		msgs = append(msgs, inference.ChatMessage{
			Role:    "system",
			Content: "--- Earlier context (compressed) ---\n" + strings.Join(summaryLines, "\n"),
		})
	}

	if pendingInput != "" {
		msgs = append(msgs, inference.ChatMessage{Role: "user", Content: pendingInput})
	}

	return msgs
}

// TrimStats holds observability data from context trimming.
type TrimStats struct {
	TotalTokens     int
	HistoryTurnsIn  int
	HistoryTurnsOut int
	MemoryTiers     map[string]int
	EmergencySummary bool
	SavedTokens     int
}

func (t TrimStats) String() string {
	return fmt.Sprintf("trim: total=%d history=%d→%d tiers=%v emergency=%v saved≈%d",
		t.TotalTokens, t.HistoryTurnsIn, t.HistoryTurnsOut, t.MemoryTiers, t.EmergencySummary, t.SavedTokens)
}

// MessageTrimmer orchestrates context trimming: budget-aware memory retrieval + BuildMessagesSafe.
type MessageTrimmer struct {
	tokenizer Tokenizer
}

// NewMessageTrimmer creates a message trimmer.
func NewMessageTrimmer(tok Tokenizer) *MessageTrimmer {
	if tok == nil {
		tok = DefaultTokenizer
	}
	return &MessageTrimmer{tokenizer: tok}
}

// Trim builds safe messages with budget-aware memory retrieval when the retriever supports it.
// Returns messages and TrimStats for observability.
func (tm *MessageTrimmer) Trim(
	ctx context.Context,
	systemPrompt string,
	recentTurns []state.Turn,
	pendingInput string,
	memoryRetriever memory.MemoryRetriever,
	toolDefs []inference.ToolDefinition,
	limits TokenLimits,
	effectiveCap int,
	log *slog.Logger,
) ([]inference.ChatMessage, TrimStats) {
	var stats TrimStats
	stats.HistoryTurnsIn = len(recentTurns)

	cap := limits.MaxInputTokens
	if effectiveCap > 0 && effectiveCap < cap {
		cap = effectiveCap
	}

	// Estimate memory budget: cap - system - history - input - tools - overhead
	toolTokens := EstimateToolTokens(toolDefs)
	systemTokens := tm.tokenizer.CountTokens(systemPrompt)
	inputTokens := tm.tokenizer.CountTokens(pendingInput)
	historyEst := tm.estimateHistoryTokens(recentTurns, limits)
	memoryBudget := cap - systemTokens - historyEst - inputTokens - toolTokens - fixedOverheadTokens - 200 // reserve
	if memoryBudget < 100 {
		memoryBudget = 100
	}

	var memoryBlock string
	if withBudget, ok := memoryRetriever.(memory.MemoryRetrieverWithBudget); ok {
		var err error
		memoryBlock, stats.MemoryTiers, err = withBudget.RetrieveWithBudget(ctx, "", pendingInput, memoryBudget)
		if err != nil {
			log.Debug("retrieve with budget failed, falling back", "err", err)
			memoryBlock, _ = memoryRetriever.Retrieve(ctx, "", pendingInput)
			stats.MemoryTiers = nil
		}
	} else {
		memoryBlock, _ = memoryRetriever.Retrieve(ctx, "", pendingInput)
	}

	messages := BuildMessagesSafe(systemPrompt, recentTurns, pendingInput, memoryBlock, toolDefs, limits, effectiveCap, tm.tokenizer, log)

	// Compute stats
	var totalTokens int
	for _, m := range messages {
		totalTokens += tm.tokenizer.CountTokens(m.Content)
	}
	totalTokens += toolTokens + fixedOverheadTokens
	stats.TotalTokens = totalTokens

	// HistoryTurnsOut: approximate from final message count
	stats.HistoryTurnsOut = len(recentTurns) // BuildMessagesSafe may have truncated
	stats.SavedTokens = stats.HistoryTurnsIn*150 - totalTokens // rough
	if stats.SavedTokens < 0 {
		stats.SavedTokens = 0
	}

	if log != nil {
		log.Debug("context_trim", "stats", stats.String())
	}

	return messages, stats
}

func (tm *MessageTrimmer) estimateHistoryTokens(turns []state.Turn, limits TokenLimits) int {
	if limits.HistoryCompress != nil && len(turns) > limits.HistoryCompress.FullTurns {
		cfg := *limits.HistoryCompress
		compressed := NewHistoryTrimmer(tm.tokenizer).Compress(turns, cfg)
		var n int
		for _, c := range compressed {
			if c.Full != nil {
				n += tm.tokenizer.CountTokens(c.Full.Input) + tm.tokenizer.CountTokens(c.Full.Thinking) + tm.tokenizer.CountTokens(c.Full.ToolCalls)
			} else {
				n += tm.tokenizer.CountTokens(c.Summary)
			}
		}
		return n
	}
	// Rough: ~150 tokens per turn
	if len(turns) > 20 {
		return 20 * 150
	}
	return len(turns) * 150
}
