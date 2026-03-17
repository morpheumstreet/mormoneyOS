# Memory Retrieval (Step 6) — Design

**Date:** 2026-03-13  
**Purpose:** 5-tier memory retrieval aligned with TS `MemoryRetriever.retrieve` + `formatMemoryBlock`. References [memory-system-5-tier.md](./memory-system-5-tier.md) for the full 5-tier architecture.

---

## 1. TS Reference

**Flow (src/agent/loop.ts):**
1. `MemoryRetriever.retrieve(sessionId, pendingInput?.content)` → `MemoryRetrievalResult`
2. `formatMemoryBlock(memories)` → string
3. `buildContextMessages(...)` → messages
4. `messages.splice(1, 0, { role: "system", content: memoryBlock })` — inject after system prompt

**5 tiers (priority order):** working > episodic > semantic > procedural > relationships

Higher-priority tiers get token budget first; unused budget rolls to the next tier.

| Tier | Table | Purpose |
|------|-------|---------|
| Working | `working_memory` | Session-scoped: goals, observations, plans, reflections, tasks, decisions |
| Episodic | `episodic_memory` | Past events: outcomes, importance; enables reflection |
| Semantic | `semantic_memory` | Facts: self, environment, financial, domain knowledge |
| Procedural | `procedural_memory` | How-to procedures: steps, success/failure counts |
| Relationship | `relationship_memory` | Entities: address, trust score, interaction count |

**Token budget:** `MemoryBudgetManager` allocates tokens across tiers; unused rolls to next.

**Data sources (TS):** Dedicated tables per tier (see [memory-system-5-tier.md §2](./memory-system-5-tier.md)).

---

## 2. Go Current State

- **Phase 3 (implemented):** 5-tier tables (schema v13); `DBMemoryRetriever` (DB-only); `BudgetAllocator`; memory block injected at index 1
- **DB-only:** No KV fallback; all five tables read from DB
- **Tools:** `remember_fact`, `recall_facts`, `forget`, `set_goal`, `complete_goal`, `save_procedure`, `recall_procedure`, `review_memory` (KV-backed)
- **Memory ingestion:** Implemented. See [memory-auto-ingestion.md](./memory-auto-ingestion.md). Opt-in via `memory.autoIngest.enabled`.

---

## 3. Architecture

| Aspect | Application |
|--------|-------------|
| **Single Responsibility** | `MemoryRetriever` retrieves; `FormatMemoryBlock` formats; loop injects |
| **Interface Segregation** | Narrow `MemoryRetriever` interface; optional in loop |
| **Open/Closed** | Add new tiers or backends without changing loop |
| **Single path** | One retrieval path, one format function; tools and retriever share KV keys |
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

`MemoryBlock` is a struct holding sections (facts, goals, procedures, etc.). Phase 1: Known Facts, Active Goals, Known Procedures. Phase 2: extended with Working, Episodic, Relationship sections. One format function; all retrievers produce `*MemoryBlock`. The retriever implementation builds `MemoryBlock`, calls `FormatMemoryBlock`, and returns the formatted string — the loop never sees `MemoryBlock`.

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

### 5.1 Data Mapping (Phase 1 KV vs Phase 2 DB)

| TS Tier | Phase 1 KV | Phase 2 DB Table | Section Label |
|---------|------------|------------------|---------------|
| working | — | `working_memory` | Working Memory |
| episodic | — | `episodic_memory` | Episodic Memory |
| semantic | `memory_facts` | `semantic_memory` | Known Facts |
| procedural | `goals`, `procedure:*` | `procedural_memory` | Active Goals / Known Procedures |
| relationship | — | `relationship_memory` | Relationships |

**DBMemoryRetriever** reads from all five tables (working, episodic, semantic, procedural, relationship). DB-only; no KV fallback. Phase 3 tables (schema v13).

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

### 5.4 Token Budget (Phase 1 vs Phase 2)

| | Phase 1 | Phase 2 |
|---|---------|---------|
| **Approach** | Simple truncation (max N facts, M goals, P procedures) | `MemoryBudgetManager` allocates tokens per tier; unused rolls to next |
| **Order** | N/A (single block) | working > episodic > semantic > procedural > relationships |
| **Rationale** | Minimal; no schema. | Ensures highest-priority context (working, episodic) is always included first. See [memory-system-5-tier.md §2](./memory-system-5-tier.md). |

Phase 1: include all facts (up to N), all pending goals, all procedures (up to M). Add proper token estimation and tiered budgeting in Phase 2.

---

## 6. Phase 2: Full 5-Tier — Implemented

Implemented as of 2026-03-14 (see [memory-system-5-tier.md §6](./memory-system-5-tier.md)):

### 6.1 Schema (5 Tables)

Add tables aligned with TS `src/state/schema.ts`:

| Table | Key Columns |
|-------|-------------|
| `working_memory` | session_id, content, content_type, priority, token_count, expires_at, source_turn |
| `episodic_memory` | session_id, event_type, summary, detail, outcome, importance, embedding_key, classification |
| `semantic_memory` | category, key, value, confidence, source, embedding_key, last_verified_at |
| `procedural_memory` | name, description, steps, success_count, failure_count, last_used_at |
| `relationship_memory` | entity_address, entity_name, relationship_type, trust_score, interaction_count, last_interaction_at, notes |

### 6.2 Retrieval

1. Implement `DBMemoryRetriever` that queries all five tables.
2. Apply **priority order:** working > episodic > semantic > procedural > relationships.
3. Implement `MemoryBudgetManager` (or equivalent) for token allocation per tier; unused rolls to next.
4. Same `MemoryRetriever` interface; same `FormatMemoryBlock`; loop unchanged.
5. `MemoryBlock` struct extended with sections for all five tiers.
6. Optional: feature flag or config to choose KV vs DB retriever.

### 6.3 Ingestion (Optional)

TS has `MemoryIngestionPipeline` that extracts from turns into tiers. Go could add (Phase 3):

- Post-turn pipeline that parses tool results and turn content.
- Writes to `working_memory` (session notes), `episodic_memory` (events), etc.
- Or continue relying on explicit tools (`remember_fact`, `save_procedure`, etc.).

### 6.4 Migration Path

| Phase | Status |
|-------|--------|
| **Phase 1** | KV-backed; no schema change. ✅ |
| **Phase 2a** | Add tables; implement `DBMemoryRetriever`, `BudgetAllocator`. ✅ |
| **Phase 2b** | Deprecate KV fallback; DB-only retrieval. ✅ |
| **Phase 3** | Wire ingestion; tool updates to write to DB. Pending |

---

## 7. File Layout

| Path | Responsibility |
|------|----------------|
| `internal/memory/retriever.go` | `MemoryRetriever` interface, `MemoryBlock` struct, `FormatMemoryBlock` |
| `internal/memory/db_retriever.go` | `DBMemoryRetriever` — queries all 5 tables (DB-only, Phase 3) |
| `internal/memory/budget.go` | `BudgetAllocator` — token allocation per tier |
| `internal/agent/loop.go` | Optional `MemoryRetriever` in options; call Retrieve; inject block |
| `internal/agent/context.go` | No change |

**Alternative:** Put retriever in `internal/agent/` if memory is agent-scoped. Prefer `internal/memory/` for clarity and future growth (ingestion, tiers).

---

## 8. Implementation Checklist

- [x] Add `KVReader` interface with `GetKV` and `ListKeysWithPrefix` (or extend existing store)
- [x] Add `ListKeysWithPrefix` to Database (or equivalent for procedure enumeration)
- [x] Create `internal/memory/` with `MemoryBlock`, `FormatMemoryBlock`, `MemoryRetriever`
- [x] ~~Implement `KVMemoryRetriever`~~ — removed; DB-only retrieval
- [x] Add `MemoryRetriever` to `LoopOptions`; wire in `cmd/run.go`
- [x] In `runOneTurnReAct`: call Retrieve, inject block at index 1 when non-empty
- [x] Tests: FormatMemoryBlock empty/non-empty; DBMemoryRetriever via integration
- [x] Update `ts-go-alignment.md` step 6 as aligned
- [x] **Phase 2:** Add 5 tables; implement `DBMemoryRetriever`; add `MemoryBudgetManager`; extend `MemoryBlock` with all five tier sections

---

## 9. Shared Constants

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

---

## 11. References

- [memory-system-5-tier.md](./memory-system-5-tier.md) — Full 5-tier architecture, schema, Phase 2 design, migration path.
- [ts-go-alignment.md](./ts-go-alignment.md) — §3.2 Tables in TS Only; §7.2 Step 6; §8.3 Memory (5-tier) readiness.
