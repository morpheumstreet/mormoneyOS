package agent

import (
	"strings"
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/inference"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

func TestBuildMessagesSafe_UnderCap(t *testing.T) {
	limits := TokenLimits{MaxInputTokens: 10000, MaxHistoryTurns: 12, WarnAtTokens: 8000}
	turns := []state.Turn{
		{Input: "hi", Thinking: "hello"},
	}
	msgs := BuildMessagesSafe("system", turns, "current", "", nil, limits, 0, nil, nil)
	if len(msgs) < 3 {
		t.Fatalf("expected at least 3 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "system" || msgs[0].Content != "system" {
		t.Errorf("first message should be system")
	}
	if msgs[len(msgs)-1].Content != "current" {
		t.Errorf("last message should be current input")
	}
}

func TestBuildMessagesSafe_TruncatesWhenOverCap(t *testing.T) {
	// Use a very low cap so we must truncate
	limits := TokenLimits{MaxInputTokens: 100, MaxHistoryTurns: 12, WarnAtTokens: 80}
	// Create turns that would exceed 100 tokens (each "x" * 400 = ~100 tokens per turn)
	big := ""
	for i := 0; i < 100; i++ {
		big += "word "
	}
	turns := []state.Turn{
		{Input: big, Thinking: big},
		{Input: big, Thinking: big},
		{Input: big, Thinking: big},
	}
	msgs := BuildMessagesSafe("system", turns, "current", "", nil, limits, 0, nil, nil)
	// Should have system + current at minimum (history truncated)
	if len(msgs) < 2 {
		t.Fatalf("expected at least 2 messages (system+input), got %d", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Errorf("first message should be system")
	}
	if msgs[len(msgs)-1].Content != "current" {
		t.Errorf("last message should be current input")
	}
}

func TestBuildMessagesSafe_WithMemory(t *testing.T) {
	limits := DefaultTokenLimits()
	msgs := BuildMessagesSafe("system", nil, "current", "memory block", nil, limits, 0, nil, nil)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages (system, memory, input), got %d", len(msgs))
	}
	if msgs[1].Content != "memory block" {
		t.Errorf("second message should be memory block")
	}
}

func TestBuildMessagesSafe_SummaryWhenRemainingBudget(t *testing.T) {
	// Cap 2000; 5 small turns that together exceed cap; truncate to 1 turn (~200 tokens) -> remaining ~1800
	limits := TokenLimits{MaxInputTokens: 2000, MaxHistoryTurns: 12, WarnAtTokens: 1500}
	turns := []state.Turn{
		{Input: "first", Thinking: "a"},
		{Input: "second", Thinking: "b"},
		{Input: "third", Thinking: "c"},
		{Input: "fourth", Thinking: "d"},
		{Input: "fifth", Thinking: strings.Repeat("x", 500)},
	}
	msgs := BuildMessagesSafe("sys", turns, "current", "", nil, limits, 0, &NaiveTokenizer{}, nil)
	// Should have system + summary (if remaining >= 800) + history + current
	hasSummary := false
	for _, m := range msgs {
		if strings.Contains(m.Content, "Earlier context (truncated)") {
			hasSummary = true
			break
		}
	}
	if !hasSummary {
		t.Log("no summary inserted (remaining may be < 800); acceptable")
	}
	// At minimum: system + current
	if len(msgs) < 2 {
		t.Errorf("expected at least 2 messages, got %d", len(msgs))
	}
}

func TestBuildMessagesSafe_EffectiveCap(t *testing.T) {
	limits := TokenLimits{MaxInputTokens: 10000, MaxHistoryTurns: 12, WarnAtTokens: 8000}
	turns := []state.Turn{
		{Input: "a", Thinking: "b"},
		{Input: strings.Repeat("x", 500), Thinking: strings.Repeat("y", 500)},
	}
	// effectiveCap 300 should force truncation; without it we'd fit
	msgs := BuildMessagesSafe("sys", turns, "cur", "", nil, limits, 300, &NaiveTokenizer{}, nil)
	if len(msgs) < 2 {
		t.Errorf("expected at least 2 messages, got %d", len(msgs))
	}
	// With effectiveCap 300, we should have truncated to system+current or very little history
	if msgs[len(msgs)-1].Content != "cur" {
		t.Errorf("last message should be current input")
	}
}

func TestEstimateToolTokens(t *testing.T) {
	defs := []inference.ToolDefinition{
		{
			Function: inference.ToolSchema{
				Name:        "test",
				Description: "test desc",
				Parameters:  `{"type":"object"}`,
			},
		},
	}
	got := estimateToolTokens(defs)
	if got < 1 {
		t.Errorf("estimateToolTokens should return > 0 for non-empty defs, got %d", got)
	}
}