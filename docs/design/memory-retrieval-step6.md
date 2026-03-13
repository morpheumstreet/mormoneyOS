# Memory Retrieval (Step 6) — Design

**Date:** 2026-03-13  
**Purpose:** Clean, DRY, SOLID design for 5-tier memory retrieval aligned with TS `MemoryRetriever.retrieve` + `formatMemoryBlock`.

---

## 1. TS Reference

**Flow (src/agent/loop.ts):**
1. `MemoryRetriever.retrieve(sessionId, pendingInput?.content)` → `MemoryRetrievalResult`
2. `formatMemoryBlock(memories)` → string
3. `buildContextMessages(...)` → messages
4. `messages.splice(1, 0, { role: "system", content: memoryBlock })` — inject after system prompt

**5 tiers (priority order):** working > episodic > semantic > procedural > relationships

**Token budget:** `MemoryBudgetManager` allocates tokens across tiers; unused rolls to next.

**Data sources (TS):** Dedicated tables: `working_memory`, `episodic_memory`, `semantic_memory`, `procedural_memory`, `relationship_memory`.

---

## 2. Go Current State

- **KV-backed:** `memory_facts`, `goals`, `procedure:<name>` (no procedure index)
- **Tools:** `remember_fact`, `recall_facts`, `forget`, `set_goal`, `complete_goal`, `save_procedure`, `recall_procedure`, `review_memory`
- **No pre-turn retrieval:** Agent must call `recall_facts` / `recall_procedure` explicitly
- **No memory block** injected into context before inference

---

## 3. Design Principles

| Principle | Application |
|-----------|-------------|
| **Single Responsibility** | `MemoryRetriever` retrieves; `FormatMemoryBlock` formats; loop injects |
| **Interface Segregation** | Narrow `MemoryRetriever` interface; optional in loop |
| **Open/Closed** | Add new tiers or backends without changing loop |
| **DRY** | One retrieval path, one format function; tools and retriever share KV keys |
| **Dependency Inversion** | Loop depends on `MemoryRetriever` interface, not concrete store |

---

## 4. Architecture

### 4.1 Interface

```go
// MemoryRetriever retrieves relevant memories for context injection (TS step 6).
// Returns a formatted block or empty string. Errors must not block the agent loop.
type MemoryRetriever interface {
    Retrieve(ctx context.Context, sessionID string, currentInput string) (block string, err error)
}
```

- **Narrow:** One method. Caller gets ready-to-inject string.
- **Optional:** Loop works without it; when nil, no memory block.
- **sessionID:** Reserved for Phase 2 session-scoped memory; Phase 1 implementations may ignore it.

### 4.2 Format (Single Source of Truth)

```go
// FormatMemoryBlock formats retrieval result into a markdown block for context.
// TS formatMemoryBlock-aligned. Returns "" when empty.
func FormatMemoryBlock(r *MemoryBlock) string
```

`MemoryBlock` is a struct holding sections (facts, goals, procedures, etc.). One format function; all retrievers produce `*MemoryBlock`. The retriever implementation builds `MemoryBlock`, calls `FormatMemoryBlock`, and returns the formatted string — the loop never sees `MemoryBlock`.

### 4.3 Loop Integration

```
BuildSystemPrompt(...)
    ↓
[Optional] memoryBlock = retriever.Retrieve(ctx, sessionID, pendingInput)
    ↓
messages = BuildContextMessages(systemPrompt, recentTurns, pendingInput)
    ↓
if memoryBlock != "" { inject at index 1 (after system) }
    ↓
inference.Chat(messages, opts)
```

- **Placement:** Same as TS — after system prompt, before conversation history.
- **Failure:** On error, `block == ""`; loop continues. No crash.

---

## 5. Phase 1: KV-Backed Retrieval (No Schema Change)

### 5.1 Data Mapping

| TS Tier      | Go Source              | KV Key(s)                    | Section Label      |
|--------------|------------------------|------------------------------|--------------------|
| semantic     | facts                  | `memory_facts`               | Known Facts        |
| procedural   | goals + procedures     | `goals`, `procedure:*`       | Active Goals / Known Procedures |
| (working)    | —                      | —                            | —                  |
| (episodic)   | —                      | —                            | —                  |
| (relationship)| —                     | —                            | —                  |

Phase 1: **Known Facts** (from `memory_facts`) + **Active Goals** (from `goals`, pending only) + **Known Procedures** (from `procedure:*` when listable). Split into distinct sections for clarity (TS separates procedural from goals; Phase 1 combines as a simplification). Procedure format for Phase 1: `- name: N steps` (description/successCount/failureCount are Phase 2).

### 5.2 KVReader Contract

Retriever needs read-only KV access. Introduce minimal interface:

```go
type KVReader interface {
    GetKV(key string) (string, bool, error)
    ListKeysWithPrefix(prefix string) ([]string, error)
}
```

`*state.Database` implements `GetKV` today; add `ListKeysWithPrefix` for procedure enumeration. Loop passes store (or a wrapper) to retriever constructor.

### 5.3 Procedure Enumeration

Procedures are stored as `procedure:<name>` = steps. No index today.

**Option A (minimal):** Add `ListKeysWithPrefix(prefix string) ([]string, error)` to Database. Query: `SELECT key FROM kv WHERE key LIKE ?`, e.g. `procedure:%`.

**Option B (defer):** Omit procedures from Phase 1 retrieval; add when procedure index exists. Start with facts + goals only.

**Recommendation:** Option A — small, reversible, enables full Phase 1.

### 5.4 Token Budget (Phase 1)

TS uses token budgets per tier. Phase 1 can skip budget: include all facts (up to N), all pending goals, all procedures (up to M). Simple truncation by count or character limit. Add proper token estimation later if needed.

---

## 6. Phase 2: Full 5-Tier (Future)

When schema adds `working_memory`, `episodic_memory`, `semantic_memory`, `procedural_memory`, `relationship_memory`:

1. Implement `DBMemoryRetriever` that queries those tables.
2. Implement `MemoryBudgetManager` (or equivalent) for token allocation.
3. Same `MemoryRetriever` interface; same `FormatMemoryBlock`; loop unchanged.
4. Optional: feature flag or config to choose KV vs DB retriever.

---

## 7. File Layout

| Path | Responsibility |
|------|----------------|
| `internal/memory/retriever.go` | `MemoryRetriever` interface, `MemoryBlock` struct, `FormatMemoryBlock` |
| `internal/memory/kv_retriever.go` | `KVMemoryRetriever` — reads facts, goals, procedures from KV |
| `internal/agent/loop.go` | Optional `MemoryRetriever` in options; call Retrieve; inject block |
| `internal/agent/context.go` | No change |

**Alternative:** Put retriever in `internal/agent/` if memory is agent-scoped. Prefer `internal/memory/` for clarity and future growth (ingestion, tiers).

---

## 8. Implementation Checklist

- [x] Add `KVReader` interface with `GetKV` and `ListKeysWithPrefix` (or extend existing store)
- [x] Add `ListKeysWithPrefix` to Database (or equivalent for procedure enumeration)
- [x] Create `internal/memory/` with `MemoryBlock`, `FormatMemoryBlock`, `MemoryRetriever`
- [x] Implement `KVMemoryRetriever` using shared keys (`memory_facts`, `goals`, `procedure:*`)
- [x] Add `MemoryRetriever` to `LoopOptions`; wire in `cmd/run.go`
- [x] In `runOneTurnReAct`: call Retrieve, inject block at index 1 when non-empty
- [x] Tests: FormatMemoryBlock empty/non-empty; KVMemoryRetriever with mock KV
- [x] Update `ts-go-alignment.md` step 6 as aligned

---

## 9. Shared Constants (DRY)

Keys used by both tools and retriever should live in one place:

```go
// internal/tools/keys.go (or internal/memory/keys.go)
const (
    MemoryFactsKey   = "memory_facts"
    GoalsKey         = "goals"
    ProcedurePrefix  = "procedure:"
)
```

Tools already use `memoryFactsKey`, `goalsKey`, `procedurePrefix`. Extract to shared package or keep in tools and have memory import from tools (avoid circular deps).

---

## 10. Risk and Rollback

- **Risk:** Low. Optional component; nil retriever = current behavior.
- **Rollback:** Remove `MemoryRetriever` from options; loop behaves as before.
- **Token impact:** Memory block adds to prompt size; monitor context length.
