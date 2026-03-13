# 5-Tier Memory System Design

**Date:** 2026-03-13  
**Purpose:** Explain the 5-tier memory architecture, why it matters, and the current Go vs TS alignment gap. References [memory-retrieval-step6.md](./memory-retrieval-step6.md) and [ts-go-alignment.md](./ts-go-alignment.md).

---

## 1. Executive Summary

The **5-tier memory system** is the TypeScript reference design for agent memory. It separates memory into five distinct tiers with dedicated tables, priority ordering, and token budgeting. The **Go implementation** has **Phase 3 (DB-only)** implemented: five dedicated tables (schema v13), `DBMemoryRetriever`, and `BudgetAllocator`. Memory retrieval is DB-only; tools write to DB tables (semantic_memory, procedural_memory, working_memory) or KV until tool migration.

**Status:** Tables exist; retrieval and formatting aligned. Memory ingestion pipeline (auto-extraction from turns) is optional Phase 3.

---

## 2. The Five Tiers (TS Reference)

| Tier | Table | Purpose | Retention |
|------|-------|---------|-----------|
| **Working** | `working_memory` | Short-term, session-scoped: goals, observations, plans, reflections, tasks, decisions | Expires; evicted by priority when full |
| **Episodic** | `episodic_memory` | Past events: what happened, outcomes, importance | Long-term; pruned by retention policy |
| **Semantic** | `semantic_memory` | Facts: self, environment, financial, domain knowledge | Persistent; category + key unique |
| **Procedural** | `procedural_memory` | How-to procedures: steps, success/failure counts | Persistent; name unique |
| **Relationship** | `relationship_memory` | Entities: address, name, type, trust score, interaction count | Persistent; entity_address unique |

**Priority order for retrieval:** working > episodic > semantic > procedural > relationships

Higher-priority tiers get token budget first; unused budget rolls to the next tier. This ensures the agent always has recent context (working) and salient past events (episodic) before general knowledge (semantic, procedural, relationships).

---

## 3. Why Each Tier Matters

### 3.1 Working Memory

- **What:** Session-scoped scratchpad. Content types: goal, observation, plan, reflection, task, decision, note, summary.
- **Why it matters:** The agent needs to hold "what I'm doing right now" and "what I just decided" across turns. Without it, each turn starts with no in-session context beyond the raw conversation history.
- **TS schema:** `session_id`, `content`, `content_type`, `priority`, `token_count`, `expires_at`, `source_turn`.

### 3.2 Episodic Memory

- **What:** Past events with summary, detail, outcome (success/failure/partial/neutral), importance, classification (strategic, productive, communication, maintenance, idle, error).
- **Why it matters:** Enables reflection ("last time I tried X it failed") and learning from experience. Supports soul reflection and alignment checks.
- **TS schema:** `session_id`, `event_type`, `summary`, `detail`, `outcome`, `importance`, `embedding_key`, `classification`.

### 3.3 Semantic Memory

- **What:** Categorized facts (self, environment, financial, agent, domain, procedural_ref, creator). Key-value with confidence and source.
- **Why it matters:** Stable knowledge the agent has learned. Maps to Go's `memory_facts` (KV) in Phase 1.
- **TS schema:** `category`, `key`, `value`, `confidence`, `source`, `embedding_key`, `last_verified_at`.

### 3.4 Procedural Memory

- **What:** Named procedures with steps, description, success_count, failure_count, last_used_at.
- **Why it matters:** Reusable how-to knowledge. Maps to Go's `procedure:*` (KV) in Phase 1; TS adds structured metadata.
- **TS schema:** `name`, `description`, `steps`, `success_count`, `failure_count`, `last_used_at`.

### 3.5 Relationship Memory

- **What:** Entities (by address): name, relationship_type, trust_score, interaction_count, last_interaction_at, notes.
- **Why it matters:** Social coherence. The agent can remember who it has interacted with, trust levels, and context for message_child, send_message, etc.
- **TS schema:** `entity_address`, `entity_name`, `relationship_type`, `trust_score`, `interaction_count`, `last_interaction_at`, `notes`.

---

## 4. Current Go State (Phase 2 Implemented)

### 4.1 What Go Has

| TS Tier | Go Implementation | Source |
|---------|-------------------|--------|
| Semantic | ✅ | `semantic_memory` (DB) + `memory_facts` (KV fallback) via `remember_fact` |
| Procedural | ✅ | `procedural_memory` (DB) + `goals`/`procedure:*` (KV fallback) |
| Working | ✅ | `working_memory` (DB) — tables ready; empty until ingestion |
| Episodic | ✅ | `episodic_memory` (DB) — tables ready; empty until ingestion |
| Relationship | ✅ | `relationship_memory` (DB) — tables ready; empty until ingestion |

**Retrieval:** `DBMemoryRetriever` reads from all five tables. `BudgetAllocator` allocates tokens per tier; unused rolls to next. Injects at index 1 (after system prompt). No KV fallback.

**Tools:** `remember_fact`, `recall_facts`, `forget`, `set_goal`, `complete_goal`, `save_procedure`, `recall_procedure`, `note_about_agent`, `review_memory` (KV-backed; ingestion to 5-tier DB is Phase 3).

### 4.2 What Go Still Lacks (Phase 3)

1. **Memory ingestion pipeline:** TS has automatic extraction from turns into tiers. Go relies on explicit tool calls (remember_fact, save_procedure, etc.); no automatic ingestion into working_memory, episodic_memory, semantic_memory, procedural_memory, relationship_memory. Until ingestion, DB tiers are populated only by future tools or manual writes.

2. **Tool updates:** `remember_fact`, `save_procedure`, etc. write to KV. Phase 3 could add dual-write or migration to DB tables.

---

## 5. Why Each Tier Matters (Addressed by Phase 2 Infrastructure)

Phase 2 adds tables and retrieval. Full benefit requires **ingestion** (Phase 3) to populate working/episodic/relationship from turns.

### 5.1 Continuity

**Addressed:** `working_memory` table exists. When ingestion writes session notes, the agent gets "what I'm working on right now" across turns.

### 5.2 Reflection and Learning

**Addressed:** `episodic_memory` table exists. When ingestion writes events with outcomes, the agent can reflect on "what happened when I did X."

### 5.3 Social Coherence

**Addressed:** `relationship_memory` table exists. When tools or ingestion write entity records, the agent gains persistent relationship context.

### 5.4 Token Efficiency

**Addressed:** `BudgetAllocator` allocates tokens per tier; unused rolls to next. Priority order (working > episodic > semantic > procedural > relationships) is enforced.

---

## 6. Design for Phase 2 (Full 5-Tier) — Implemented

Phase 2 implemented as of 2026-03-14:

### 6.1 Schema

Add five tables aligned with TS `src/state/schema.ts`:

- `working_memory` — session_id, content, content_type, priority, token_count, expires_at, source_turn
- `episodic_memory` — session_id, event_type, summary, detail, outcome, importance, embedding_key, classification
- `semantic_memory` — category, key, value, confidence, source, embedding_key, last_verified_at
- `procedural_memory` — name, description, steps, success_count, failure_count, last_used_at
- `relationship_memory` — entity_address, entity_name, relationship_type, trust_score, interaction_count, last_interaction_at, notes

### 6.2 Retrieval

1. Implement `DBMemoryRetriever` that queries all five tables.
2. Apply priority order: working > episodic > semantic > procedural > relationships.
3. Implement `MemoryBudgetManager` (or equivalent) for token allocation per tier; unused rolls to next.
4. Same `MemoryRetriever` interface; same `FormatMemoryBlock`; loop unchanged.
5. Optional: feature flag or config to choose KV vs DB retriever.

### 6.3 Ingestion (Optional)

TS has `MemoryIngestionPipeline` that extracts from turns into tiers. Go could add:

- Post-turn pipeline that parses tool results and turn content.
- Writes to working_memory (session notes), episodic_memory (events), etc.
- Or continue relying on explicit tools; ingestion can be Phase 3.

### 6.4 Migration Path

| Phase | Status |
|-------|--------|
| **Phase 1** | KV-backed; no schema change. ✅ |
| **Phase 2a** | Add tables; implement DBMemoryRetriever; BudgetAllocator. ✅ |
| **Phase 2b** | Deprecate KV fallback; DB-only retrieval. ✅ |
| **Phase 3** | Wire ingestion; tool updates to write to DB tables. Pending |

---

## 7. References

- [memory-retrieval-step6.md](./memory-retrieval-step6.md) — Retrieval interface, Phase 2 implementation, loop integration.
- [ts-go-alignment.md](./ts-go-alignment.md) — §3.1 5-tier tables; §7.2 Step 6; §8.3 Memory (5-tier) readiness.
- `src/state/schema.ts` — TS table definitions.
- `internal/memory/retriever.go` — MemoryRetriever interface, MemoryBlock, FormatMemoryBlock.
- `internal/memory/db_retriever.go` — DBMemoryRetriever (DB-only, Phase 3).
- `internal/memory/budget.go` — BudgetAllocator (token allocation per tier).
