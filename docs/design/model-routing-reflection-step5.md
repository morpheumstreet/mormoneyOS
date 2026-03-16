# Model Routing + Self-Critique / Reflection — Design (Step 5)

**Date:** 2026-03-17  
**Purpose:** Document model tier selection (fast/normal/strong) and Reflexion-style self-critique on impactful turns. Enables cost-efficient inference (90% cheap model) while reserving strong models for high-stakes decisions, and feeds critique output into the automatic memory pipeline for continuous improvement.

---

## 1. Problem

- **Cost:** Using a strong model for every turn is expensive; most turns are routine.
- **Risk:** Financial moves (transfer_credits, fund_child) and high-token contexts need stronger reasoning.
- **Learning:** The agent has no structured way to reflect on its actions and improve over time.

---

## 2. Solution Overview

- **Model Router:** Select fast/normal/strong tier per turn based on `DecisionContext` (tokens, risk, money impact).
- **Reflection Engine:** Run cheap critique on impactful turns; output (lessons, recommendations) feeds memory ingestion.
- **Zero duplication:** Reuses `Factory`, `ClientHolder`, `TokenLimits`, metrics, automatic ingestion, versioned prompts.
- **Configurable:** YAML/JSON routing section; hot-reload via existing `ClientHolder` + `Router.Reload()`.

---

## 3. Architecture

### 3.1 Files

| File | Purpose |
|------|---------|
| `internal/inference/types.go` | `ModelTier`, `DecisionContext`, `RoutingRiskLevel` |
| `internal/inference/router.go` | `ModelRouter`, `Select()`, `ClientForReflection()` |
| `internal/agent/reflection.go` | `ReflectionEngine`, `CritiqueTurn()`, `Reflection` |
| `internal/inference/router_test.go` | Router tests |
| `internal/prompts/templates/v1/critique.tmpl` | Versioned critique prompt |
| `internal/tools/mutating.go` | `IsMoneyMovingTool()` |
| `internal/agent/loop.go` | Integration: `router.Select`, `reflectionEngine.CritiqueTurn` |
| `internal/memory/service.go` | `IngestReflection()` |
| `internal/agent/metrics.go` | `RecordRoutingStrong`, `RecordRoutingFast`, `RecordCritique` |

### 3.2 Core Types

| Type | Location | Purpose |
|------|----------|---------|
| `ModelTier` | `inference/types.go` | `TierFast`, `TierNormal`, `TierStrong` |
| `DecisionContext` | `inference/types.go` | `TokensUsed`, `RiskLevel`, `HasMoneyImpact`, `TurnPhase` |
| `RoutingRiskLevel` | `inference/types.go` | `RiskLow`, `RiskMedium`, `RiskHigh` |
| `Reflection` | `agent/reflection.go` | `SuccessScore`, `Lessons`, `MemoryRecommendations` |
| `ReflectionData` | `memory/ingester.go` | Input for `IngestReflection` |

### 3.3 Routing Logic

```go
// Select decides tier
if TokensUsed >= StrongThresholdTokens → TierStrong
if ForceStrongOnMoneyMove && HasMoneyImpact → TierStrong
if RiskLevel == RiskHigh → TierStrong
else → DefaultTier (from config, usually "normal")
```

### 3.4 Client Selection

- **TierNormal:** Uses `holder.Client()` (main configurable client).
- **TierFast:** First model from `cfg.Models` (sorted by priority).
- **TierStrong:** Last model from `cfg.Models` (sorted by priority).
- **Fallback:** When `cfg.Models` is empty, uses holder for all tiers.

---

## 4. Configuration

### 4.1 Config Schema

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
|-------|---------|-------------|
| `defaultTier` | `"normal"` | Tier when no upgrade rules apply |
| `strongThresholdTokens` | `3500` | Use strong model when input tokens ≥ this |
| `forceStrongOnMoneyMove` | `true` | Use strong model when `HasMoneyImpact` |
| `reflectionTier` | `"fast"` | Tier for critique calls (always cheap) |

### 4.2 HasMoneyImpact Heuristic

`HasMoneyImpact` is set when `pendingInput` contains keywords: `transfer`, `fund`, `send` (case-insensitive). Used because tool calls are unknown until after inference.

---

## 5. Integration

### 5.1 Loop Flow

1. **Before inference:** Build `DecisionContext`, call `router.Select(ctx, dc)`, use returned client.
2. **Inference:** Call `client.Chat(ctx, messages, opts)` (unchanged).
3. **After tool execution:** When `anyMutatingToolExecuted`, run `CritiqueTurn`, then `IngestReflection` if `MemoryIngester` implements `ReflectionIngester`.

### 5.2 Critique Prompt

Versioned `critique.tmpl` (v1) expects JSON output:

```json
{
  "success_score": 0.0–1.0,
  "lessons": ["lesson1", "lesson2"],
  "memory_recommendations": ["rec1"]
}
```

### 5.3 Memory Ingestion

`IngestReflection` converts `ReflectionData` to a synthetic `TurnData` and calls `Ingester.Ingest`, so critique output flows through the same extraction pipeline as regular turns.

---

## 6. Metrics (expvar)

| Metric | Purpose |
|--------|---------|
| `routing_strong_total` | Turns routed to strong model |
| `routing_fast_total` | Turns routed to fast model |
| `critique_total` | Critique calls executed |

---

## 7. Tools

| Function | Location | Purpose |
|----------|----------|---------|
| `IsMoneyMovingTool(name)` | `internal/tools/mutating.go` | True for `transfer_credits`, `fund_child` |

---

## 8. Related Documents

- [token-caps-truncation.md](./token-caps-truncation.md) — Token limits, `CountMessagesTokens`
- [memory-auto-ingestion.md](./memory-auto-ingestion.md) — Automatic ingestion pipeline
- [inference-hot-reload.md](./inference-hot-reload.md) — `ClientHolder`, `Router.Reload`
