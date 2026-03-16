# Token Caps + Truncation — Design

**Date:** 2026-03-17  
**Purpose:** Hard token caps and truncation to avoid provider prefill limits (e.g. Groq ~6k–8k). Eliminates "empty response (prefill limit ~6k tokens)" errors by ensuring total input tokens ≤ safe threshold before every LLM call.

---

## 1. Problem

Many providers (Groq, some OpenRouter models) enforce prefill caps (~6k–8k tokens). When the agent sends more input tokens than the provider supports, the model returns empty or truncated responses. The agent loop had no token budgeting; long history + memory + system prompt could exceed caps.

---

## 2. Solution Overview

- **Single source of truth** for token counting and truncation in `internal/agent`.
- **Before every inference call:** count tokens, enforce cap, truncate history if needed.
- **Priority order:** system prompt > memory block > recent history (newest first) > current input.
- **Configurable limits** via `automaton.json`; fail gracefully with logging instead of crashing.

---

## 3. Architecture

### 3.1 Files

| File | Purpose |
|------|---------|
| `internal/agent/token.go` | `Tokenizer` interface, `NaiveTokenizer`, `TokenLimits` |
| `internal/agent/prompt.go` | `BuildMessagesSafe` — cap enforcement + truncation |
| `internal/agent/trim.go` | `HistoryTrimmer`, `MessageTrimmer`, `CompressedTurn` |
| `internal/agent/loop.go` | Integration: `MessageTrimmer.Trim` or `BuildMessagesSafe` |
| `internal/memory/select.go` | `TieredMemorySelector`, `TieredMemoryRetriever` |
| `internal/agent/token_test.go` | Tokenizer and limits tests |
| `internal/agent/prompt_test.go` | `BuildMessagesSafe` tests |
| `internal/agent/trim_test.go` | HistoryTrimmer, MessageTrimmer tests |

### 3.2 Tokenizer Interface

```go
type Tokenizer interface {
    CountTokens(text string) int
}
```

- **Production:** `NaiveTokenizer` — ~4 chars/token, ~5–10% error for English. Zero dependencies.
- **Future:** `tiktoken-go/tokenizer` for OpenAI-style accuracy when needed.

### 3.3 Token Limits

```go
type TokenLimits struct {
    MaxInputTokens   int                   // Safe threshold before truncation (default 5500)
    MaxHistoryTurns int                    // Max history turns to keep when truncating (default 12)
    WarnAtTokens    int                    // Log warning when input exceeds this (default 4500)
    HistoryCompress *HistoryTrimmerConfig  // Optional; rule-based history compression
}
```

Defaults target Groq and similar providers. Override via config. When `HistoryCompress` is set and `len(turns) > FullTurns`, history is compressed before truncation (see [context-trimming-stage2.md](./context-trimming-stage2.md)).

### 3.4 BuildMessagesSafe Flow

1. **Phase 1 — Build full messages:** system + memory block + history (from `BuildContextMessages`) + current input.
2. **Phase 2 — Count tokens:** messages + tool schemas (`estimateToolTokens`) + fixed overhead (50).
3. **Phase 3 — Enforce cap:**
   - If total ≤ `MaxInputTokens`: return as-is (warn if ≥ `WarnAtTokens`).
   - Else: truncate history. Try keeping `MaxHistoryTurns` down to 0 turns (newest first). First fit under cap wins.
   - Fallback: system + memory + current input only (rare).

### 3.5 Truncation Priority

| Priority | Component | Truncation |
|----------|-----------|------------|
| 1 | System prompt | Never truncated |
| 2 | Memory block | Never truncated |
| 3 | History turns | Newest first; drop oldest until under cap |
| 4 | Current input | Never truncated (always last message) |

History is truncated by **complete turns** (user + assistant pairs) to preserve coherence.

---

## 4. Config

### 4.1 automaton.json

```json
{
  "maxInputTokens": 5500,
  "maxHistoryTurns": 12,
  "warnAtTokens": 4500
}
```

Snake_case aliases supported: `max_input_tokens`, `max_history_turns`, `warn_at_tokens`.

### 4.2 types.AutomatonConfig

```go
MaxInputTokens  int  // default 5500
MaxHistoryTurns int  // default 12
WarnAtTokens    int  // default 4500
```

### 4.3 LoopConfig

```go
TokenLimits *TokenLimits  // optional; nil = DefaultTokenLimits()
```

Populated in `cmd/run.go` via `tokenLimitsFromConfig(cfg)`.

---

## 5. Loop Integration

### 5.1 History Fetch

- Previously: `GetRecentTurns(5)`.
- Now: `GetRecentTurns(50)` (or `MaxHistoryTurns` if larger) for context building.
- Wakeup summaries still use 5 turns.

### 5.2 Message Building

- Previously: `BuildContextMessages` + manual memory injection at index 1.
- Now: `BuildMessagesSafe(systemPrompt, recentTurns, pendingInput, memoryBlock, toolDefs, limits, tok, log)`.

### 5.3 Observability

- **Warn:** when input ≥ `WarnAtTokens`.
- **Warn:** when truncation triggered (`tokens`, `cap`).
- **Info:** after truncation (`original_tokens`, `final_tokens`, `kept_turns`).
- **Warn:** fallback to system+memory+input only.
- **Debug:** before inference (`model`, `messages` count).

---

## 6. Testing

| Test | File | Purpose |
|------|------|---------|
| `TestNaiveTokenizer_*` | token_test.go | Empty, short, approximate, long text |
| `TestDefaultTokenLimits` | token_test.go | Default values |
| `TestTokenLimits_WithOverrides` | token_test.go | Override behavior |
| `TestBuildMessagesSafe_UnderCap` | prompt_test.go | No truncation when under cap |
| `TestBuildMessagesSafe_TruncatesWhenOverCap` | prompt_test.go | Truncation when over cap |
| `TestBuildMessagesSafe_WithMemory` | prompt_test.go | Memory block injection |
| `TestEstimateToolTokens` | prompt_test.go | Tool schema token estimate |

---

## 7. Design Principles

| Principle | Application |
|-----------|-------------|
| **DRY** | Single `BuildMessagesSafe`; one token counting path |
| **Clean** | No duplication; centralize in agent package |
| **Solid** | Defensive (nil tok → DefaultTokenizer), configurable, testable |
| **Observable** | Logs at warn/info/debug for truncation and cap proximity |

---

## 8. Phase 2 (Implemented)

- **Tiktoken:** `TiktokenTokenizer` using `github.com/tiktoken-go/tokenizer` (cl100k_base). `DefaultTokenizer` uses tiktoken when available, else `NaiveTokenizer`.
- **Summarization:** When truncating with remaining budget ≥ 800 tokens, inserts heuristic summary of dropped turns (`buildDroppedTurnsSummary`) before history.
- **Tier-aware pruning:** `ContextLimitForModel` in `LoopConfig`; per-model `ContextLimit` from `cfg.Models`; `effectiveCap` passed to `BuildMessagesSafe`.
- **expvar:** `agent_input_tokens_total`, `agent_truncations_total` published via `internal/agent/metrics.go`.

---

## 9. Phase 3 — Context Trimming Stage 2 (Implemented)

See [context-trimming-stage2.md](./context-trimming-stage2.md) for full design.

- **HistoryTrimmer:** Rule-based compression — full last 6 turns, summarize turns 7–20, drop older. Heuristic summaries from tool results or thinking.
- **TieredMemorySelector:** Per-tier soft/hard caps (working 1800/2200, episodic 1500/2000, etc.). `TieredMemoryRetriever` implements `MemoryRetrieverWithBudget`.
- **MessageTrimmer:** Orchestrator — estimates memory budget, calls `RetrieveWithBudget` when supported, then `BuildMessagesSafe`. Returns `TrimStats` for observability.
- **Loop:** Uses `MessageTrimmer.Trim` when `memoryRetriever` is set; `cmd/run.go` uses `TieredMemoryRetriever` for budget-aware memory.
