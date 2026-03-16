# Simulation / Backtest Mode — Design

**Date:** 2026-03-17  
**Purpose:** Deterministic simulation and backtesting to validate agent stability, memory pipeline, and token handling before real-money runs. Reuses the exact same `agent.Loop`, `MemoryService`, `PromptBuilder`, and `TokenTrimmer` as production.

---

## 1. Philosophy & Non-Goals

- **Solid**: 100% deterministic by default, reproducible seeds, chaos toggle, crash-proof.
- **Dry**: Zero code duplication — reuses `agent.Loop`, `MemoryService`, `PromptBuilder`, `TokenTrimmer`, metrics, config.
- **Clean**: Same style as `internal/memory/` and `internal/agent/`: small focused files, interfaces, config structs, expvar metrics.
- **Non-goal**: Full market simulator engine — we replay real historical data + mocks only.

---

## 2. CLI Integration

```bash
moneyclaw sim --days=30 --speed=100x --chaos=medium --seed=42 --report=html
moneyclaw sim --config=sim-prod.json --strategies=trend-following
```

New file: `cmd/sim.go` (subcommand under root, like `run.go`).

---

## 3. Package Structure

```
internal/simulation/
├── config.go          # SimulationConfig + defaults
├── interfaces.go      # MarketReplayProvider, InferenceResponder, Clock
├── simulator.go       # Simulator (main orchestrator)
├── runner.go          # SimulationRunner (virtual clock + loop driver)
├── replay.go          # MarketReplayProvider + CSV/Conway implementations
├── chaos.go           # ChaosInjector (API failures, price shocks, latency)
├── metrics.go         # SimMetricsCollector (extends agent/metrics)
├── report.go          # Reporter (JSON + embedded HTML)
├── mock/              # mock_inferencer.go, mock_market.go, mock_clock.go
└── simulation_test.go # integration tests (5000-turn runs)
```

---

## 4. Core Interfaces

```go
// internal/simulation/interfaces.go
type MarketReplayProvider interface {
    NextTick(ctx context.Context) (Tick, error)  // price, volume, news, etc.
    ResetToDay(start time.Time)
}

type InferenceResponder interface {  // for deterministic LLM replies
    Respond(turn *agent.TurnInput) (*agent.TurnOutput, error)
}

type Clock interface {  // virtual time for heartbeat/scheduler
    Now() time.Time
    Advance(d time.Duration)
}
```

All existing components stay untouched:
- `agent.Loop` receives the same `*LoopConfig` (with `SimMode: true` when needed)
- `MemoryService` uses in-memory or test DB
- `PromptBuilder` and `TokenTrimmer` used exactly as in real runs
- `InferenceClient` swapped for mock when in sim mode

---

## 5. Flow

```go
sim := NewSimulator(cfg, db)
runner := NewSimulationRunner(sim)

for day := 0; day < cfg.Days; day++ {
    replay.ResetToDay(startDate)
    for tick := range replay.Ticks() {
        runner.AdvanceClock(tick.Time)
        result := loop.RunOneTurn(ctx, agentState)  // same loop as production
        sim.metrics.RecordTurn(result)
        ingester.IngestTurn(result)  // auto-ingestion still runs
    }
    consolidator.Tick()  // background consolidation still active
}
reporter.Generate("sim-results/")
```

---

## 6. Metrics & Reporting

Reuses `expvar` + adds:
- `sim_turns_total`, `sim_crashes_total`, `sim_pnl_usd`, `sim_token_usage_peak`
- `memory_growth_per_day`, `trim_events_total`, `ingestion_candidates_processed`

Reporter outputs:
- `sim-report.json` (full trace)
- `sim-report.html` (embedded dashboard using existing DashOS assets)

---

## 7. Config (Viper + JSON)

```json
{
  "simulation": {
    "days": 30,
    "speedMultiplier": 100,
    "chaosLevel": "medium",
    "seed": 42,
    "marketDataSource": "csv:./data/binance-2025.csv",
    "reportFormat": "html"
  }
}
```

Defaults in `simulation/config.go` (same style as `memory/config.go`).

---

## 8. Chaos Engine

Configurable injection rates:
- API timeouts (10%)
- Price flash crashes (5%)
- LLM empty responses (rare, to test trimmer)
- Random memory eviction

All logged with seed for reproducibility.

---

## 9. Integration Points (Zero Breaking Changes)

- `cmd/run.go` → extract `buildAgentLoop(cfg)` into `internal/agent/factory.go` (tiny refactor)
- `internal/state/database.go` → already has test migrations
- `internal/memory/service.go` → optional `SetMockStore` for sim (or use test DB)
- `agent/loop.go` → `if cfg.SimMode { useMockClockAndResponder }`
- New design doc: this file

---

## 10. Implementation Order

1. `config.go` + interfaces + mocks (1 day)
2. `replay.go` + `chaos.go` (2 days)
3. `runner.go` + `simulator.go` (3 days)
4. `metrics.go` + `report.go` + CLI (2 days)
5. Tests + docs + CI (2 days)
