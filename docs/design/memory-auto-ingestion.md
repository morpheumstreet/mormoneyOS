# Automatic Memory Ingestion & Consolidation — Design

**Date:** 2026-03-17  
**Purpose:** Document the automatic memory ingestion pipeline that turns every ReAct turn into structured, tiered knowledge without manual `remember_*` tools. References [memory-system-5-tier.md](./memory-system-5-tier.md) and [memory-retrieval-step6.md](./memory-retrieval-step6.md).

---

## 1. Executive Summary

**Goal:** Automatically turn every ReAct turn (reasoning + action + observation) into structured, tiered knowledge **without** manual `remember_*` tools, while keeping context clean, token usage bounded, and enabling true long-term learning.

**Design principles:**
- Extend only `internal/memory/` (no new top-level packages)
- Reuse existing: `BudgetAllocator`, `DBMemoryRetriever`, metrics, trim logic, prompt templates, `TurnResult`, `state.DB`, heartbeat lifecycle
- Single responsibility + tiny files
- Non-blocking (never slows the main loop)
- Fully observable + configurable + testable
- Idempotent and crash-safe

**Status:** Implemented. Opt-in via `memory.autoIngest.enabled` in config.

---

## 2. Architecture Overview

### 2.1 Two-Phase Pipeline

| Phase | Component | When | Purpose |
|-------|-----------|------|---------|
| **1. Ingester** | `ingester.go` | After every turn | Light extraction via cheap model; stores raw `Extraction` as `IngestCandidate` |
| **2. Consolidator** | `consolidator.go` | Background ticker (e.g. every 12 min) | Batch classification, deduplication, migration to 5-tier tables |

### 2.2 Package Structure

```
internal/memory/
├── budget.go           // existing; extend with pruning helpers if needed
├── db_retriever.go     // existing
├── retriever.go        // existing
├── types.go            // Extraction, MemoryItem, Tier, Fact, Episode, Procedure
├── ingester.go         // per-turn extraction
├── consolidator.go     // background batch worker
├── service.go          // MemoryService facade
├── ingestion_test.go   // tests
└── (optional) queue.go // for durable candidate queue if needed
```

---

## 3. Core Types

### 3.1 Extraction (`types.go`)

```go
type MemoryTier string
const (
    TierWorking     MemoryTier = "working"
    TierEpisodic    MemoryTier = "episodic"
    TierSemantic    MemoryTier = "semantic"
    TierProcedural  MemoryTier = "procedural"
    TierRelationship MemoryTier = "relationship"
)

type Extraction struct {
    Facts         []Fact
    Episodes      []Episode
    Procedures    []Procedure
    Relationships []RelationshipUpdate
    Importance    float64  // 0.0–1.0 for pruning
}

type Fact struct {
    Category   string
    Key        string
    Value      string
    Confidence float64
}

type Episode struct {
    EventType  string
    Summary    string
    Detail     string
    Outcome    string
    Importance float64
}

type Procedure struct {
    Name        string
    Steps       []string
    SuccessRate float64
}

type RelationshipUpdate struct {
    EntityAddress string
    EntityName    string
    Type          string
    TrustDelta    float64
}
```

### 3.2 TurnData (Ingestion Input)

```go
type TurnData struct {
    TurnID      string
    Timestamp   string
    SessionID   string
    Input       string
    InputSource string
    Thinking    string
    ToolCalls   string  // JSON array of {name, result, error}
}
```

---

## 4. Phase 1: Light Ingester

### 4.1 Flow

1. Called after **every** turn (both `finishReason stop` and tool-call paths)
2. Uses **cheapest/fastest model** (configurable via `cheapModel`)
3. Tiny prompt: "Extract max 3 facts, 1 procedure, 1 episode + relationship changes. Return strict JSON."
4. Parses response into `Extraction`
5. Stores raw JSON in `ingest_candidates` table
6. Target: < 800 tokens, < 1s latency

### 4.2 Extraction Prompt

Rules: max 3 facts, 1 procedure, 1 episode, relationship changes only if entities interacted. `importance` 0–1. Empty arrays ok. Returns strict JSON (no markdown).

### 4.3 Non-Blocking

- Errors are logged; loop continues
- No retries in hot path
- Idempotent: duplicate turn IDs overwrite or skip per design

---

## 5. Phase 2: Background Consolidator

### 5.1 Flow

1. Ticker every N minutes (default 12; configurable)
2. Fetches up to M unprocessed candidates (default 40)
3. For each: parse `Extraction`, apply to 5-tier tables
4. Marks candidates as processed
5. Uses `BudgetAllocator` + trim logic for pruning (future)

### 5.2 Tier Mapping

| Extraction Field | Target Table | Notes |
|------------------|--------------|-------|
| `Facts` | `semantic_memory` | UPSERT by category+key |
| `Episodes` | `episodic_memory` | INSERT (events are unique) |
| `Procedures` | `procedural_memory` | UPSERT by name; merge success/failure counts |
| `Relationships` | `relationship_memory` | UPSERT by entity_address |

### 5.3 Deduplication

- **Semantic:** `ON CONFLICT(category, key) DO UPDATE`
- **Procedural:** `ON CONFLICT(name) DO UPDATE` with merged counts
- **Relationship:** `ON CONFLICT(entity_address) DO UPDATE` with merged trust/interaction
- **Episodic:** No dedup; each event is distinct

---

## 6. Public API

### 6.1 MemoryService

```go
type MemoryService struct {
    ingester     *Ingester
    consolidator *Consolidator
    db           *state.Database
    config       MemoryConfig
}

func NewMemoryService(cfg MemoryConfig, db *state.Database, inferenceClient inference.Client, log *slog.Logger) *MemoryService

func (s *MemoryService) IngestTurn(ctx context.Context, turn *TurnData) error
func (s *MemoryService) StartBackground(ctx context.Context) error
func (s *MemoryService) Stop()
func (s *MemoryService) SetMetrics(m IngestMetricsRecorder)
```

### 6.2 MemoryConfig

```go
type MemoryConfig struct {
    AutoIngestEnabled       bool
    CheapModel              string
    ConsolidationIntervalMin int
    MaxCandidatesPerBatch   int
}
```

---

## 7. Integration

### 7.1 Loop Integration (`internal/agent/loop.go`)

```go
// MemoryIngester interface
type MemoryIngester interface {
    IngestTurn(ctx context.Context, turn *memory.TurnData) error
}

// After InsertTurn (both paths):
if l.memoryIngester != nil {
    _ = l.memoryIngester.IngestTurn(ctx, &memory.TurnData{
        TurnID: turnID, Timestamp: ts, SessionID: "", Input: pendingInput,
        InputSource: inputSource, Thinking: thinking, ToolCalls: toolCallsJSON,
    })
}
```

### 7.2 Startup (`cmd/run.go`)

1. Create `MemoryService` when `memory.autoIngest.enabled` is true
2. Pass as `MemoryIngester` in `LoopOptions`
3. Call `memSvc.StartBackground(ctx)` after context creation
4. Call `memSvc.Stop()` in defer for clean shutdown

---

## 8. Schema

### 8.1 ingest_candidates Table (schema v14)

```sql
CREATE TABLE ingest_candidates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    turn_id TEXT NOT NULL,
    extraction_json TEXT NOT NULL,
    importance REAL NOT NULL DEFAULT 0,
    processed INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_ingest_candidates_processed ON ingest_candidates(processed);
CREATE INDEX idx_ingest_candidates_created ON ingest_candidates(created_at);
```

### 8.2 5-Tier Insert Methods

`state.Database` now has:
- `InsertIngestCandidate`
- `GetUnprocessedIngestCandidates`
- `MarkIngestCandidatesProcessed`
- `InsertWorkingMemory`
- `InsertEpisodicMemory`
- `InsertSemanticMemory`
- `InsertProceduralMemory`
- `InsertRelationshipMemory`

---

## 9. Configuration

### 9.1 automaton.json

```json
{
  "memory": {
    "autoIngest": {
      "enabled": true,
      "cheapModel": "gpt-4o-mini",
      "consolidationIntervalMinutes": 12,
      "maxCandidatesPerBatch": 40
    }
  }
}
```

### 9.2 Defaults

| Field | Default |
|-------|---------|
| `enabled` | false (opt-in) |
| `cheapModel` | `gpt-4o-mini` |
| `consolidationIntervalMinutes` | 12 |
| `maxCandidatesPerBatch` | 40 |

---

## 10. Metrics (expvar)

| Name | Description |
|------|--------------|
| `memory_ingest_turns_total` | Count of turns ingested |
| `memory_consolidated_items` | Count of items written to 5-tier tables |
| `memory_pruned_count` | Count of items pruned (future) |
| `memory_extraction_latency_ms` | Extraction latency in milliseconds |

---

## 11. Why This Design

- **Solid:** Non-blocking, circuit-breaker ready, idempotent, crash-safe, testable with mocked LLM
- **DRY:** Reuses prompt system, budget, trim, retriever, metrics, state.DB, heartbeat patterns
- **Clean:** Tiny focused files, clear interfaces, follows existing Go style
- **Future-proof:** Easy to plug in embeddings for semantic search without touching the loop

---

## 12. Rollout Phases

1. **Phase 1 (done):** `types.go` + `ingester.go` + `IngestTurn` + loop hook
2. **Phase 2 (done):** `consolidator.go` + background worker
3. **Phase 3 (done):** Metrics, config, full integration
4. **Future:** Pruning via `BudgetAllocator`, embeddings for semantic search
