package agent

import (
	"sync"

	"github.com/tiktoken-go/tokenizer"
)

// Tokenizer counts tokens in text. Implementations may be approximate (naive) or accurate (tiktoken).
type Tokenizer interface {
	CountTokens(text string) int
}

// NaiveTokenizer estimates tokens as ~4 chars per token + overhead. ~5–10% error for English.
// Zero dependencies; suitable for truncation and budgeting.
type NaiveTokenizer struct{}

// CountTokens returns an approximate token count for the given text.
func (n *NaiveTokenizer) CountTokens(text string) int {
	if text == "" {
		return 0
	}
	// ~4 chars per token for typical English; add small overhead for punctuation/short words
	base := len(text) / 4
	// Adjust for likely token boundaries (spaces, newlines)
	spaces := 0
	for _, r := range text {
		if r == ' ' || r == '\n' {
			spaces++
		}
	}
	// Rough: more spaces = slightly fewer tokens (words chunk together)
	adj := spaces / 8
	if base > adj {
		base -= adj
	}
	if base < 1 {
		return 1
	}
	return base
}

// TiktokenTokenizer uses OpenAI's cl100k_base encoding (GPT-4/3.5). Accurate; ~4MB vocab.
type TiktokenTokenizer struct {
	enc tokenizer.Codec
}

// NewTiktokenTokenizer creates a tiktoken-based tokenizer. Returns nil and uses NaiveTokenizer on error.
func NewTiktokenTokenizer() *TiktokenTokenizer {
	enc, err := tokenizer.Get(tokenizer.Cl100kBase)
	if err != nil {
		return nil
	}
	return &TiktokenTokenizer{enc: enc}
}

// CountTokens returns the token count using cl100k_base.
func (t *TiktokenTokenizer) CountTokens(text string) int {
	if t == nil || t.enc == nil {
		return (&NaiveTokenizer{}).CountTokens(text)
	}
	ids, _, err := t.enc.Encode(text)
	if err != nil {
		return len(text) / 4 // fallback
	}
	return len(ids)
}

var (
	tiktokenOnce sync.Once
	tiktokenTok  Tokenizer
)

// TiktokenTokenizerOrDefault returns TiktokenTokenizer if available, else NaiveTokenizer.
func TiktokenTokenizerOrDefault() Tokenizer {
	tiktokenOnce.Do(func() {
		if tt := NewTiktokenTokenizer(); tt != nil {
			tiktokenTok = tt
		} else {
			tiktokenTok = &NaiveTokenizer{}
		}
	})
	return tiktokenTok
}

// DefaultTokenizer is the production tokenizer (tiktoken when available, else naive).
var DefaultTokenizer Tokenizer = TiktokenTokenizerOrDefault()

// TokenLimits holds configurable limits for input truncation (Groq ~6k–8k prefill caps).
type TokenLimits struct {
	MaxInputTokens   int                    // Safe threshold before truncation (default 5500)
	MaxHistoryTurns  int                    // Max history turns to keep when truncating (default 12)
	WarnAtTokens     int                    // Log warning when input exceeds this (default 4500)
	HistoryCompress *HistoryTrimmerConfig // Optional; when set, use rule-based history compression before truncation
}

// DefaultTokenLimits returns sensible defaults for Groq and similar providers.
func DefaultTokenLimits() TokenLimits {
	return TokenLimits{
		MaxInputTokens:  5500,
		MaxHistoryTurns: 12,
		WarnAtTokens:    4500,
	}
}

// WithOverrides applies non-zero overrides from cfg.
func (t TokenLimits) WithOverrides(maxInput, maxHistory, warnAt int) TokenLimits {
	if maxInput > 0 {
		t.MaxInputTokens = maxInput
	}
	if maxHistory > 0 {
		t.MaxHistoryTurns = maxHistory
	}
	if warnAt > 0 {
		t.WarnAtTokens = warnAt
	}
	return t
}
