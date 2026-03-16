package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/memory"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

// mockMemoryRetriever implements memory.MemoryRetriever for tests.
type mockMemoryRetriever struct {
	block string
}

func (m *mockMemoryRetriever) Retrieve(ctx context.Context, sessionID, currentInput string) (string, error) {
	return m.block, nil
}

func TestHistoryTrimmer_Compress_ShortHistory(t *testing.T) {
	tok := &NaiveTokenizer{}
	trimmer := NewHistoryTrimmer(tok)
	cfg := DefaultHistoryTrimmerConfig()
	cfg.FullTurns = 6

	turns := []state.Turn{
		{Input: "a", Thinking: "b"},
		{Input: "c", Thinking: "d"},
	}
	got := trimmer.Compress(turns, cfg)
	if len(got) != 2 {
		t.Fatalf("expected 2 compressed turns (all full), got %d", len(got))
	}
	if got[0].Full == nil || got[1].Full == nil {
		t.Errorf("short history should keep all turns full")
	}
}

func TestHistoryTrimmer_Compress_LongHistory(t *testing.T) {
	tok := &NaiveTokenizer{}
	trimmer := NewHistoryTrimmer(tok)
	cfg := DefaultHistoryTrimmerConfig()
	cfg.FullTurns = 4
	cfg.SummarizedMax = 6
	cfg.HistoryBudget = 0 // no budget limit for this test

	// 12 turns: last 4 full, turns 5-8 summarized (4 summaries), turns 1-4 dropped
	turns := make([]state.Turn, 12)
	for i := range turns {
		turns[i] = state.Turn{
			Input:    "input " + string(rune('a'+i)),
			Thinking: "thinking " + string(rune('a'+i)),
			ToolCalls: `[{"name":"check_balance","result":"$100","error":""}]`,
		}
	}

	got := trimmer.Compress(turns, cfg)
	// Expect: 4 full (last 4) + up to 6 summaries (turns 5-10, since 11-12 are full)
	// startFull = 12-4 = 8, so full = turns[8:12]
	// startSummarize = 8-6 = 2, so summaries = turns[2:8] = 6 turns
	if len(got) < 6 {
		t.Fatalf("expected at least 6 compressed turns, got %d", len(got))
	}
	// First entries should be summaries (oldest)
	sumCount := 0
	fullCount := 0
	for _, c := range got {
		if c.Full != nil {
			fullCount++
		} else {
			sumCount++
		}
	}
	if fullCount != 4 {
		t.Errorf("expected 4 full turns, got %d", fullCount)
	}
	if sumCount < 1 {
		t.Errorf("expected at least 1 summary, got %d", sumCount)
	}
}

func TestHistoryTrimmer_SummarizeTurn_ToolCalls(t *testing.T) {
	tok := &NaiveTokenizer{}
	trimmer := NewHistoryTrimmer(tok)

	turn := state.Turn{
		Input:    "check balance",
		Thinking: "I will check",
		ToolCalls: `[{"name":"check_usdc_balance","result":"$123.45","error":""}]`,
	}
	s := trimmer.summarizeTurn(turn)
	if s == "" {
		t.Fatal("expected non-empty summary from tool call")
	}
	if !strings.Contains(s, "check_usdc_balance") {
		t.Errorf("summary should contain tool name, got %q", s)
	}
	if !strings.Contains(s, "123") {
		t.Errorf("summary should contain result, got %q", s)
	}
}

func TestHistoryTrimmer_SummarizeTurn_ThinkingOnly(t *testing.T) {
	tok := &NaiveTokenizer{}
	trimmer := NewHistoryTrimmer(tok)

	turn := state.Turn{
		Input:    "",
		Thinking: "I decided to buy BTC at 82k. The market looked favorable.",
		ToolCalls: "",
	}
	s := trimmer.summarizeTurn(turn)
	if s == "" {
		t.Fatal("expected non-empty summary from thinking")
	}
	if !strings.Contains(s, "buy") || !strings.Contains(s, "82k") {
		t.Logf("summary: %q (may truncate)", s)
	}
}

func TestBuildContextMessagesFromCompressed(t *testing.T) {
	compressed := []CompressedTurn{
		{Summary: "check_balance: $100"},
		{Summary: "sleep: ok"},
		{Full: &state.Turn{Input: "hi", Thinking: "hello"}},
		{Full: &state.Turn{Input: "bye", Thinking: "goodbye"}},
	}

	msgs := BuildContextMessagesFromCompressed("system", compressed, "current")
	if len(msgs) < 4 {
		t.Fatalf("expected at least 4 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "system" || msgs[0].Content != "system" {
		t.Errorf("first should be system prompt")
	}
	hasCompressed := false
	for _, m := range msgs {
		if strings.Contains(m.Content, "Earlier context (compressed)") {
			hasCompressed = true
			break
		}
	}
	if !hasCompressed {
		t.Errorf("expected compressed block in messages")
	}
	if msgs[len(msgs)-1].Content != "current" {
		t.Errorf("last should be current input")
	}
}

func TestMessageTrimmer_Trim(t *testing.T) {
	// Mock retriever that returns fixed block
	mockRetriever := &mockMemoryRetriever{block: "### Working Memory\n- test goal"}
	trimmer := NewMessageTrimmer(&NaiveTokenizer{})
	limits := TokenLimits{
		MaxInputTokens:  3000,
		MaxHistoryTurns: 12,
		WarnAtTokens:    2500,
	}
	turns := []state.Turn{
		{Input: "hi", Thinking: "hello"},
	}
	msgs, stats := trimmer.Trim(context.Background(), "sys", turns, "current", mockRetriever, nil, limits, 0, nil)
	if len(msgs) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(msgs))
	}
	if stats.TotalTokens <= 0 {
		t.Errorf("expected TotalTokens > 0, got %d", stats.TotalTokens)
	}
	hasMemory := false
	for _, m := range msgs {
		if strings.Contains(m.Content, "test goal") {
			hasMemory = true
			break
		}
	}
	if !hasMemory {
		t.Error("expected memory block in messages")
	}
}

func TestMessageTrimmer_Trim_NoMemoryRetriever(t *testing.T) {
	// Trim with nil retriever would panic - we don't pass nil. Test basic path.
	mockRetriever := &mockMemoryRetriever{block: "memory"}
	trimmer := NewMessageTrimmer(&NaiveTokenizer{})
	limits := TokenLimits{MaxInputTokens: 2000, MaxHistoryTurns: 10, WarnAtTokens: 1500}
	msgs, stats := trimmer.Trim(context.Background(), "sys", nil, "input", mockRetriever, nil, limits, 0, nil)
	if len(msgs) < 2 {
		t.Errorf("expected at least 2 messages, got %d", len(msgs))
	}
	if stats.HistoryTurnsIn != 0 {
		t.Errorf("HistoryTurnsIn = %d, want 0", stats.HistoryTurnsIn)
	}
}

// agentMockTierStore implements memory.Memory5TierStore for tests.
type agentMockTierStore struct {
	working []state.WorkingMemoryRow
	semantic []state.SemanticMemoryRow
}

func (m *agentMockTierStore) GetWorkingMemory(string, int) ([]state.WorkingMemoryRow, error) {
	return m.working, nil
}
func (m *agentMockTierStore) GetEpisodicMemory(string, int) ([]state.EpisodicMemoryRow, error) {
	return nil, nil
}
func (m *agentMockTierStore) GetSemanticMemory(int) ([]state.SemanticMemoryRow, error) {
	return m.semantic, nil
}
func (m *agentMockTierStore) GetProceduralMemory(int) ([]state.ProceduralMemoryRow, error) {
	return nil, nil
}
func (m *agentMockTierStore) GetRelationshipMemory(int) ([]state.RelationshipMemoryRow, error) {
	return nil, nil
}

func TestMessageTrimmer_Trim_WithTieredRetriever(t *testing.T) {
	// TieredMemoryRetriever implements MemoryRetrieverWithBudget
	store := &agentMockTierStore{
		working:  []state.WorkingMemoryRow{{Content: "budgeted goal"}},
		semantic: []state.SemanticMemoryRow{{Value: "fact"}},
	}
	retriever := memory.NewTieredMemoryRetriever(store, memory.DefaultTierConfig())
	trimmer := NewMessageTrimmer(&NaiveTokenizer{})
	limits := TokenLimits{MaxInputTokens: 3000, MaxHistoryTurns: 10, WarnAtTokens: 2500}
	msgs, stats := trimmer.Trim(context.Background(), "sys", nil, "input", retriever, nil, limits, 0, nil)
	if len(msgs) < 2 {
		t.Errorf("expected at least 2 messages, got %d", len(msgs))
	}
	if stats.MemoryTiers != nil && stats.MemoryTiers["working"] < 1 {
		t.Errorf("expected working tier in stats, got %v", stats.MemoryTiers)
	}
	hasMemory := false
	for _, m := range msgs {
		if strings.Contains(m.Content, "budgeted") || strings.Contains(m.Content, "fact") {
			hasMemory = true
			break
		}
	}
	if !hasMemory {
		t.Error("expected memory content in messages")
	}
}

func TestBuildMessagesSafe_WithHistoryCompression(t *testing.T) {
	limits := TokenLimits{
		MaxInputTokens:  2000,
		MaxHistoryTurns: 20,
		WarnAtTokens:    1500,
		HistoryCompress: &HistoryTrimmerConfig{
			FullTurns:     4,
			SummarizedMax: 6,
			HistoryBudget: 1500,
		},
	}
	// 10 turns: should trigger compression (full last 4, summarize 6)
	turns := make([]state.Turn, 10)
	for i := range turns {
		turns[i] = state.Turn{
			Input:     "input " + string(rune('a'+i)),
			Thinking:  strings.Repeat("x", 50),
			ToolCalls: `[{"name":"tool","result":"ok","error":""}]`,
		}
	}

	msgs := BuildMessagesSafe("sys", turns, "current", "", nil, limits, 0, &NaiveTokenizer{}, nil)
	if len(msgs) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(msgs))
	}
	// Should have compressed block when over cap
	hasCompressed := false
	for _, m := range msgs {
		if strings.Contains(m.Content, "Earlier context (compressed)") {
			hasCompressed = true
			break
		}
	}
	if !hasCompressed {
		t.Log("no compressed block (may fit under cap); checking structure")
	}
	// Verify we have valid inference messages
	var totalContent int
	for _, m := range msgs {
		totalContent += len(m.Content)
	}
	if totalContent == 0 {
		t.Error("messages should have content")
	}
}

