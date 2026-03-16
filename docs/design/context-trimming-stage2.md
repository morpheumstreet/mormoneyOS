# Context Trimming Stage 2 — History + Memory Truncation

**Date:** 2026-03-17  
**Purpose:** Design for aggressive history compression and tiered memory selection to prevent prefill/token-limit crashes while preserving useful context for financial reasoning. Builds on [token-caps-truncation.md](./token-caps-truncation.md) (Step #1).

---

## 1. Goal

Prevent prefill/token-limit crashes in a production-useful way (not just hard-cut), while preserving as much useful context as reasonably possible for financial reasoning.

---

## 2. Core Principles

| Principle | Application |
|-----------|-------------|
| **Fail-safe first** | Never send > 5500 tokens (safety margin under ~6k prefill) |
| **Prioritize usefulness** | Preserve signal (recent decisions, open positions, risk rules, learned facts) over noise (old chit-chat) |
| **Dry & predictable** | Deterministic rules + minimal LLM calls |
| **Observable** | Every truncation/summary logs what was kept/dropped + approximate tokens saved |
| **Composable** | Builds on Step #1 (token counting exists) |

---

## 3. Components

| Component | Location | Responsibility | LLM usage? |
|-----------|----------|----------------|------------|
| `TokenCounter` | `internal/agent/token.go` | Exact or fast approximate token count (tiktoken compatible) | No |
| `MessageTrimmer` | `internal/agent/trim.go` | Main orchestrator: trims in stages until under budget | No |
| `HistoryTrimmer` | `internal/agent/trim.go` | Rule-based compression of conversation history | No |
| `TieredMemorySelector` | `internal/memory/select.go` | Selects memory entries by per-tier soft/hard caps | No |
| `EmergencySummarizer` | — | Cheap LLM call when still over limit after rule-based trimming | Yes (future) |
| `BudgetConfig` | `TokenLimits`, `TierConfig` | Per-tier soft/hard limits + global max | No |

---

## 4. HistoryTrimmer — Rule-Based Compression

**File:** `internal/agent/trim.go`

Compresses conversation history without LLM calls. Highest token savings for long sessions.

### 4.1 Rules

| Turn range | Treatment |
|------------|-----------|
| Last N turns (default 6) | Keep full (input + thinking + tool results) |
| Turns N+1 to N+M (default 7–20) | Summarize to one-line (action + result) |
| Older than N+M | Drop (assume consolidated to semantic/episodic) |

### 4.2 Summarization Heuristics

- **Prefer:** Tool call + result (e.g. `check_usdc_balance: $123.45`)
- **Fallback:** First meaningful line of thinking
- **Last resort:** Truncated input

### 4.3 Config

```go
type HistoryTrimmerConfig struct {
    FullTurns     int  // keep last N full (default 6)
    SummarizedMax int  // max turns to summarize (default 14)
    HistoryBudget int  // max tokens for history block (0 = no limit)
}
```

Set via `TokenLimits.HistoryCompress`. Enabled by default when `len(turns) > FullTurns`.

---

## 5. TieredMemorySelector — Per-Tier Soft/Hard Caps

**File:** `internal/memory/select.go`

Selects memory entries respecting per-tier budgets. Priority order: working > episodic > semantic > procedural > relationship.

### 5.1 Default Tier Limits

| Tier | Soft cap | Hard cap | Selection rule |
|------|----------|----------|----------------|
| Working | 1800 | 2200 | Keep all (most critical) |
| Episodic | 1500 | 2000 | Newest first; drop oldest until under |
| Semantic | 1200 | 1500 | Highest priority / most recent |
| Procedural | 800 | 1000 | Keep until budget exhausted |
| Relationship | 600 | 800 | Keep until budget exhausted |

### 5.2 API

```go
func (s *TieredMemorySelector) Select(ctx, sessionID, input string, totalBudget int) (SelectResult, error)
// SelectResult: Block string, Used int, Stats map[string]int
```

### 5.3 TieredMemoryRetriever

Implements `MemoryRetriever` and `MemoryRetrieverWithBudget`:

```go
Retrieve(ctx, sessionID, input) (block string, err error)
RetrieveWithBudget(ctx, sessionID, input string, budget int) (block string, stats map[string]int, err error)
```

When `MessageTrimmer` has a budget, it uses `RetrieveWithBudget` to fit memory within the remaining context.

---

## 6. MessageTrimmer — Orchestrator

**File:** `internal/agent/trim.go`

Orchestrates full context trimming: budget-aware memory retrieval + `BuildMessagesSafe`.

### 6.1 Flow

1. **Estimate memory budget:** `cap - system - history - input - tools - overhead`
2. **Get memory:** If retriever implements `MemoryRetrieverWithBudget`, call `RetrieveWithBudget(budget)`; else `Retrieve()`
3. **Build messages:** Call `BuildMessagesSafe` with the memory block (which applies history compression and truncation)
4. **Log stats:** `TrimStats` (TotalTokens, HistoryTurnsIn/Out, MemoryTiers, etc.)

### 6.2 TrimStats (Observability)

```go
type TrimStats struct {
    TotalTokens     int
    HistoryTurnsIn  int
    HistoryTurnsOut int
    MemoryTiers     map[string]int  // tier -> count included
    EmergencySummary bool
    SavedTokens     int
}
```

Logged at debug: `context_trim: total=4980 history=18→6 tiers=map[working:5 episodic:12] ...`

---

## 7. Trimming Stages (Applied in Order)

1. **Global hard cap** — If after all stages still > 5500 → drop to minimal context (system + current input).
2. **Tier priority & soft caps** — `TieredMemorySelector.Select(budgetLeft)` applies per-tier limits.
3. **History compression** — `HistoryTrimmer.Compress()`: full last 6, summarize 7–20, drop older.
4. **Emergency summarization** — (Future) Cheap LLM call when still over; replace history + lower-priority memory with summary.
5. **Final assembly & validation** — Re-count tokens; log structured stats.

---

## 8. Loop Integration

```go
// In runOneTurnReAct
if l.memoryRetriever != nil {
    trimmer := NewMessageTrimmer(DefaultTokenizer)
    messages, _ = trimmer.Trim(ctx, systemPrompt, recentTurnsForContext, pendingInput,
        l.memoryRetriever, toolDefs, limits, effectiveCap, l.log)
} else {
    messages = BuildMessagesSafe(..., "", ...)
}
```

`cmd/run.go` uses `TieredMemoryRetriever` (replacing `DBMemoryRetriever`) so budget-aware retrieval is active.

---

## 9. Config (Future)

Planned `automaton.json` structure:

```json
{
  "context": {
    "maxPrefillSafe": 5500,
    "maxPrefillHard": 6000,
    "tiers": {
      "working": { "soft": 1800, "hard": 2200 },
      "episodic": { "soft": 1500, "hard": 2000 }
    },
    "emergencySummarizer": {
      "enabled": true,
      "model": "llama-3.1-8b-instant",
      "targetTokens": 400
    }
  }
}
```

Currently tier limits are in code (`DefaultTierConfig()`); config loading is future work.

---

## 10. Success Criteria

| Criterion | Status |
|-----------|--------|
| Agent survives 300+ consecutive turns without prefill error | Target |
| Context size stabilizes (does not grow forever) | Target |
| Emergency summary triggered < 5% of turns after 100 turns | Target |
| Important facts (e.g. "avoid leverage >3x") reach LLM even on turn 250 | Target |

---

## 11. Related Documents

- [token-caps-truncation.md](./token-caps-truncation.md) — Step #1: token counting, `BuildMessagesSafe`
- [memory-system-5-tier.md](./memory-system-5-tier.md) — 5-tier memory architecture
- [memory-retrieval-step6.md](./memory-retrieval-step6.md) — Memory retrieval in agent loop
