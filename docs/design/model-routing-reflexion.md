# Model Routing & Reflexion Layer — Design

**Date:** 2026-03-17  
**Purpose:** Lightweight, deterministic routing that selects the optimal inference model per agent turn, paired with a post-turn self-critique (Reflexion) step. Achieves cost reduction (≥90% of turns on fast/cheap tiers), elevated reasoning when financial risk is high, and continuous self-improvement via critique → memory ingestion. References [context-trimming-stage2.md](./context-trimming-stage2.md), [token-caps-truncation.md](./token-caps-truncation.md), and [memory-auto-ingestion.md](./memory-auto-ingestion.md).

---

## 1. Goal

- **Cost:** Run ≥90% of turns on fast/cheap inference tiers.
- **Quality:** Reserve strong models for high-stakes decisions (money movement, high risk, complex context).
- **Learning:** Feed structured critiques into the automatic memory ingestion pipeline for long-term self-improvement.

---

## 2. Core Principles

| Principle | Application |
|-----------|-------------|
| **Deterministic** | Pure rule-based routing; no ML, no external service |
| **Auditable** | Every routing decision is explicit and overridable in config |
| **Zero disruption** | Sits between context-trimming and LLM call; no changes to loop core, retriever, or prompt builder |
| **Token-aware** | Never route Strong if it would violate the 5500-token safe cap |
| **Single responsibility** | Router decides only; ReflectionEngine critiques only |

---

## 3. Architecture

### 3.1 Placement in Turn Lifecycle

1. Existing: messages built, history compressed, memory block injected.
2. **Routing step:** DecisionContext assembled → ModelRouter.Select returns client.
3. Existing: LLM call proceeds with routed client (drop-in).
4. Existing: action executed.
5. **Reflection step (optional):** If impact detected, ReflectionEngine runs critique (Fast tier) → result ingested via MemoryService.IngestReflection.
6. Turn ends; metrics updated; memory consolidation proceeds as before.

### 3.2 Components

| Component | Location | Responsibility |
|-----------|----------|----------------|
| `ModelRouter` | `internal/inference/router.go` | Selects client per turn from DecisionContext |
| `ReflectionEngine` | `internal/agent/reflection.go` | Runs self-critique on impactful turns; always uses Fast tier |
| `DecisionContext` | `internal/inference/types.go` | Transient snapshot for routing (tokens, risk, money impact, phase) |
| `RoutingConfig` | `internal/types/types.go` | Config: default tier, escalation thresholds, reflection tier |
| `MemoryService.IngestReflection` | `internal/memory/service.go` | Feeds critique output into ingestion pipeline |

---

## 4. Model Tiers

Three stable, configurable tiers mapped to `cfg.Models` (by priority):

| Tier | Use case | Model selection |
|------|----------|-----------------|
| **Fast** | Routine turns, all critiques | First model in sorted list (lowest priority index) |
| **Normal** | Balanced default | Holder's default client (when no router or fallback) |
| **Strong** | High-stakes only | Last model in sorted list (highest priority index) |

Strong tier is the exception, never the rule.

---

## 5. DecisionContext

A transient, read-only snapshot built from already-available data. Created once per turn and discarded after routing.

| Field | Source |
|-------|--------|
| `TokensUsed` | Token caps layer (`CountMessagesTokens`) |
| `RiskLevel` | Policy engine; see evolution path below |
| `HasMoneyImpact` | Heuristic: input contains "transfer", "fund", "send" |
| `TurnPhase` | "planning", "action", "reflection" |
| `Uncertainty` | Optional; from prompt analysis (0–1) |

**Current implementation:** Loop builds `DecisionContext` in `runOneTurnReAct` before calling `ModelRouter.Select`. Risk level is currently `RiskLow`; money impact is keyword-based.

**RiskLevel evolution path:**
- **Short-term:** Expose a simple `RiskEvaluator` interface in `internal/policy/` (stub returning Low/Medium/High based on tool name + amount threshold).
- **Medium-term:** Let the policy engine inject real risk scores into `DecisionContext` when it matures.

**Uncertainty (future):** Derive a basic proxy from prompt analysis (e.g., count of "I think", "maybe", "unsure" in previous thought) or from critique `success_score` trend.

---

## 6. Routing Policy

Pure rule-based. Escalation triggers are explicit and overridable in config.

### 6.1 Default Escalation Rules (decideTier)

| Condition | Result |
|-----------|--------|
| `TokensUsed >= StrongThresholdTokens` | Strong |
| `ForceStrongOnMoneyMove && HasMoneyImpact` | Strong |
| `RiskLevel == RiskHigh` | Strong |
| Else | DefaultTier (typically "normal") |

### 6.2 Token Safety (Merge Blocker)

**Must-have before first merge to dev/main.** Never route Strong if `TokensUsed` exceeds the prefill safe cap — otherwise the router may attempt a doomed call that hits prefill limits again.

Add in `decideTier` after computing tier:

```text
if tier == Strong && dc.TokensUsed > cfg.TokenCapForStrong { tier = fallbackTier }
```

Config `tokenCapForStrong` (default 5500) is added to `RoutingConfig` (see §8.2 Additional Config Fields). This guard costs almost nothing and protects against config mis-tunings.

### 6.3 Fallback

When `clientForTier` returns nil (e.g. no models in catalog), router falls back to `holder.Client()`.

---

## 7. Reflection Engine

Dedicated component that runs **after** action execution, only when the turn meets configurable impact thresholds.

### 7.1 When It Runs

- **Current:** When `anyMutatingToolExecuted` is true (tool executed that mutates state).
- **Configurable:** `reflectionOnAllTurns` (default false) — when true, run critique on every turn for debugging/learning phases.
- **Future:** Soft frequency cap (e.g., max 1 critique every 5 turns) to avoid critique spam on high-activity loops. Impact filters, confidence thresholds.

### 7.2 Output

Structured JSON: `success_score`, `lessons`, `memory_recommendations`. Parsed by `parseCritiqueResponse` in `reflection.go`.

### 7.3 Integration with Memory

- `ReflectionEngine.CritiqueTurn` returns `*Reflection` (primary type name; use consistently in code/docs).
- Loop checks `l.memoryIngester.(ReflectionIngester)` and calls `IngestReflection(ctx, rd)`.
- `MemoryService.IngestReflection` formats critique as a synthetic turn and passes to `Ingester.Ingest` → `ingest_candidates` → consolidator.

No new storage or DB changes; reuses the existing automatic ingestion pipeline. **Naming:** Prefer `Reflection` as the canonical type; `CritiqueTurnData` and `ReflectionData` are input/output variants — keep `IngestReflection` as the method name.

---

## 8. Configuration

### 8.1 RoutingConfig (types.AutomatonConfig.Routing)

```json
{
  "routing": {
    "defaultTier": "normal",
    "strongThresholdTokens": 3500,
    "forceStrongOnMoneyMove": true,
    "reflectionTier": "fast"
  }
}
```

| Field | Default | Description |
|-------|--------|-------------|
| `defaultTier` | "normal" | Tier when no escalation triggers |
| `strongThresholdTokens` | 3500 | Escalate to Strong when tokens ≥ this |
| `forceStrongOnMoneyMove` | true | Use Strong for transfer/fund/send |
| `reflectionTier` | "fast" | Tier for critique calls (always cheap) |

### 8.2 Additional Config Fields

| Field | Default | Description |
|-------|---------|-------------|
| `tokenCapForStrong` | 5500 | Never route Strong above this; used by §6.2 guard |
| `reflectionOnAllTurns` | false | Run critique on every turn (debugging/learning) |
| `reflectionFrequencyCap` | 0 | Max critiques per N turns; 0 = no cap |

### 8.3 Future Extensions

- `reflectionEnabled`, `reflectionImpactFilters`, confidence thresholds
- Environment overrides (consistent with current config handling)

---

## 9. Loop Integration

### 9.1 Model Router

```go
// In runOneTurnReAct, after message building:
client := l.inference
if l.modelRouter != nil {
    tokensUsed := CountMessagesTokens(messages, toolDefs, DefaultTokenizer)
    hasMoneyImpact := strings.Contains(strings.ToLower(pendingInput), "transfer") || ...
    dc := inference.DecisionContext{
        TokensUsed:     tokensUsed,
        RiskLevel:      inference.RiskLow,
        HasMoneyImpact: hasMoneyImpact,
        TurnPhase:      "action",
    }
    if routed, err := l.modelRouter.Select(ctx, dc); err == nil && routed != nil {
        client = routed
        model = client.GetDefaultModel()
    }
}
// LLM call uses client
```

### 9.2 Reflection Engine

```go
// After InsertTurn, when mutating tools executed:
if l.reflectionEngine != nil && anyMutatingToolExecuted {
    ref, err := l.reflectionEngine.CritiqueTurn(ctx, &CritiqueTurnData{...})
    if err == nil && ref != nil {
        rd := &memory.ReflectionData{...}
        if ri, ok := l.memoryIngester.(ReflectionIngester); ok {
            _ = ri.IngestReflection(ctx, rd)
        }
    }
}
```

### 9.3 Startup (cmd/run.go)

- `ModelRouter` created when `cfg.Routing != nil || len(cfg.Models) > 0`.
- `ReflectionEngine` created when `modelRouter != nil`; passed as `ReflectionEngine` in `LoopOptions`.
- Both are optional; loop works without them.

---

## 10. Observability

### 10.1 Metrics (expvar)

| Name | Description |
|------|-------------|
| `routing_strong_total` | Count of turns routed to Strong tier |
| `routing_fast_total` | Count of turns routed to Fast tier |
| `routing_strong_reason_tokens` | Strong escalations due to token threshold |
| `routing_strong_reason_money` | Strong escalations due to money impact |
| `routing_strong_reason_risk` | Strong escalations due to risk level |
| `critique_total` | Count of reflection invocations |
| `critique_success_score_avg` | Rolling average of parsed `success_score` |

**Future:** `routing_normal_total`, `routing_decision_reason`, `reflection_invocation_rate`.

### 10.2 Logging

- Debug: `routing: strong (tokens)`, `routing: strong (money impact)`, `routing: strong (risk)`.
- Critique failures: `critique prompt build failed`, `critique inference failed`.

### 10.3 Safeguards (Future)

- Hard fallback to safest model on repeated Strong-tier failures (circuit-breaker).
- Full audit logging of every routing decision.

---

## 11. Hot-Reload

- `ModelRouter.Reload()` clears cached fast/strong clients; next `Select` recreates from config.
- Holder pattern: `InferenceClientHolder.Reload(cfg)` already supports config hot-reload.
- Routing config values support hot-reload when config file is updated.

---

## 12. Success Criteria

| Criterion | Status |
|-----------|--------|
| ≥90% of turns on Fast/Normal tier | Target (metrics in place) |
| Strong tier only for money/risk/high-token turns | Implemented |
| Critique runs on mutating turns | Implemented |
| Critique output ingested into memory | Implemented |
| Token-aware: never Strong if > 5500 tokens | Merge blocker (§6.2) |
| Circuit-breaker on Strong failures | Future |

---

## 13. Observed Results (Post-Deploy)

*To be filled once live:* After ~1000 turns, document tier distribution %, avg critique score, cost savings estimate. Use `routing_*_total` and `critique_success_score_avg` metrics.

---

## 14. Related Documents

- [context-trimming-stage2.md](./context-trimming-stage2.md) — History compression, tiered memory, MessageTrimmer
- [token-caps-truncation.md](./token-caps-truncation.md) — Token counting, BuildMessagesSafe, 5500 cap
- [memory-auto-ingestion.md](./memory-auto-ingestion.md) — IngestTurn, IngestReflection, consolidator
- [memory-retrieval-step6.md](./memory-retrieval-step6.md) — Memory retrieval in agent loop
