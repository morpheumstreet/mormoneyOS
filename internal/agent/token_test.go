package agent

import (
	"testing"
)

func TestNaiveTokenizer_Empty(t *testing.T) {
	tok := &NaiveTokenizer{}
	if got := tok.CountTokens(""); got != 0 {
		t.Errorf("CountTokens(\"\") = %d, want 0", got)
	}
}

func TestNaiveTokenizer_Short(t *testing.T) {
	tok := &NaiveTokenizer{}
	// ~4 chars per token; "hello" (5 chars) -> at least 1
	if got := tok.CountTokens("hello"); got < 1 {
		t.Errorf("CountTokens(\"hello\") = %d, want >= 1", got)
	}
}

func TestNaiveTokenizer_Approximate(t *testing.T) {
	tok := &NaiveTokenizer{}
	// "Hello world" = 11 chars -> ~2-3 tokens
	got := tok.CountTokens("Hello world")
	if got < 1 || got > 5 {
		t.Errorf("CountTokens(\"Hello world\") = %d, expected roughly 2-3", got)
	}
}

func TestNaiveTokenizer_LongText(t *testing.T) {
	tok := &NaiveTokenizer{}
	// 400 chars -> ~100 tokens
	text := ""
	for i := 0; i < 100; i++ {
		text += "word "
	}
	got := tok.CountTokens(text)
	// 500 chars, many spaces -> expect ~100-150
	if got < 50 || got > 200 {
		t.Errorf("CountTokens(500 chars) = %d, expected ~100-150", got)
	}
}

func TestDefaultTokenLimits(t *testing.T) {
	limits := DefaultTokenLimits()
	if limits.MaxInputTokens != 5500 {
		t.Errorf("MaxInputTokens = %d, want 5500", limits.MaxInputTokens)
	}
	if limits.MaxHistoryTurns != 12 {
		t.Errorf("MaxHistoryTurns = %d, want 12", limits.MaxHistoryTurns)
	}
	if limits.WarnAtTokens != 4500 {
		t.Errorf("WarnAtTokens = %d, want 4500", limits.WarnAtTokens)
	}
}

func TestTokenLimits_WithOverrides(t *testing.T) {
	limits := DefaultTokenLimits().WithOverrides(6000, 20, 5000)
	if limits.MaxInputTokens != 6000 {
		t.Errorf("MaxInputTokens = %d, want 6000", limits.MaxInputTokens)
	}
	if limits.MaxHistoryTurns != 20 {
		t.Errorf("MaxHistoryTurns = %d, want 20", limits.MaxHistoryTurns)
	}
	if limits.WarnAtTokens != 5000 {
		t.Errorf("WarnAtTokens = %d, want 5000", limits.WarnAtTokens)
	}
}

func TestTokenLimits_WithOverrides_ZeroPreservesDefault(t *testing.T) {
	limits := DefaultTokenLimits().WithOverrides(0, 0, 0)
	if limits.MaxInputTokens != 5500 {
		t.Errorf("MaxInputTokens = %d, want 5500 (default)", limits.MaxInputTokens)
	}
	if limits.MaxHistoryTurns != 12 {
		t.Errorf("MaxHistoryTurns = %d, want 12 (default)", limits.MaxHistoryTurns)
	}
}

func TestTiktokenTokenizer(t *testing.T) {
	tok := NewTiktokenTokenizer()
	if tok == nil {
		t.Skip("tiktoken not available")
	}
	got := tok.CountTokens("Hello world")
	if got < 1 || got > 5 {
		t.Errorf("CountTokens(\"Hello world\") = %d, expected ~2-3", got)
	}
	// "Hello world" is typically 2 tokens in cl100k
	if got := tok.CountTokens(""); got != 0 {
		t.Errorf("CountTokens(\"\") = %d, want 0", got)
	}
}
