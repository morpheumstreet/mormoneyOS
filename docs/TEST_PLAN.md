# mormoneyOS-go Test Plan

Test plan to validate all features and functions per [mormoneyOS design](../mormoneyOS/docs/design).

---

## 1. Test Categories

| Category | Scope | Type |
|----------|-------|------|
| Unit | Internal packages | `go test ./internal/...` |
| Integration | CLI + config + DB | Manual / scripted |
| E2E | Full bootstrap + run | Manual |
| Security | Policy engine, path protection | Unit + integration |

---

## 2. Config (`internal/config`)

| ID | Test Case | Input | Expected |
|----|-----------|-------|----------|
| C1 | `GetAutomatonDir` with default | no env | `$HOME/.automaton` |
| C2 | `GetAutomatonDir` with override | `AUTOMATON_DIR=/tmp/auto` | `/tmp/auto` |
| C3 | `GetConfigPath` | — | `{automatonDir}/automaton.json` |
| C4 | `ResolvePath` with `~` | `~/foo` | `$HOME/foo` |
| C5 | `ResolvePath` without `~` | `/abs/path` | `/abs/path` |
| C6 | `Load` when no file | missing config | `nil, nil` |
| C7 | `Load` when invalid JSON | malformed file | error |
| C8 | `Load` merges with defaults | partial config | defaults filled for missing fields |
| C9 | `Load` treasury merge | `treasuryPolicy.maxSingleTransferCents: 1000` | overrides default 5000 |
| C10 | `Save` creates dir | new config | `~/.automaton` created, file written |
| C11 | `Save` round-trip | save then load | config matches |

---

## 3. Types (`internal/types`)

| ID | Test Case | Input | Expected |
|----|-----------|-------|----------|
| T1 | `DefaultTreasuryPolicy` | — | `MaxSingleTransferCents=5000`, `MinReserveCents=100` |
| T2 | `AgentState` constants | — | `setup`, `waking`, `running`, `sleeping`, `low_compute`, `critical`, `dead` |
| T3 | `SurvivalTier` constants | — | `high`, `normal`, `low_compute`, `critical`, `dead` |
| T4 | `RiskLevel` constants | — | `safe`, `caution`, `dangerous`, `forbidden` |

---

## 4. Conway Credits (`internal/conway`)

| ID | Test Case | Input (cents) | Expected Tier |
|----|-----------|---------------|---------------|
| CR1 | `TierFromCreditsCents` high | 501 | `high` |
| CR2 | `TierFromCreditsCents` normal | 51 | `normal` |
| CR3 | `TierFromCreditsCents` low_compute | 11 | `low_compute` |
| CR4 | `TierFromCreditsCents` critical | 0 | `critical` |
| CR5 | `TierFromCreditsCents` dead | -1 | `dead` |
| CR6 | Boundary high/normal | 500 | `normal` |
| CR7 | Boundary normal/low | 50 | `low_compute` |
| CR8 | Boundary low/critical | 10 | `critical` |

---

## 5. Policy Engine (`internal/agent`)

| ID | Test Case | Input | Expected |
|----|-----------|-------|----------|
| P1 | `ToolArgsHash` deterministic | same args | same hash |
| P2 | `ToolArgsHash` different args | different args | different hash |
| P3 | `ValidationRule` empty name | `tool.Name=""` | deny, "tool name is empty" |
| P4 | `ValidationRule` whitespace name | `tool.Name="  "` | deny |
| P5 | `ValidationRule` valid name | `tool.Name="exec"` | allow |
| P6 | `PathProtectionRule` protected write | `path="/x/constitution"` | deny |
| P7 | `PathProtectionRule` protected read | `path="/x/api-key"` | deny |
| P8 | `PathProtectionRule` safe path | `path="/tmp/foo"` | allow |
| P9 | `PathProtectionRule` no path arg | `Args={}` | allow (skip) |
| P10 | `AuthorityRule` creator | `source="creator"` | allow |
| P11 | `AuthorityRule` self | `source="self"` | allow |
| P12 | `AuthorityRule` external + dangerous | `source="external"`, `risk=dangerous` | deny |
| P13 | `AuthorityRule` external + safe | `source="external"`, `risk=safe` | allow |
| P14 | `PolicyEngine.Evaluate` first deny wins | ValidationRule denies | returns false, ValidationRule reason |
| P15 | `PolicyEngine.Evaluate` all allow | all rules pass | returns true, "" |
| P16 | `CreateDefaultRules` | — | 3 rules (Validation, PathProtection, Authority) |

---

## 6. State / Database (`internal/state`)

| ID | Test Case | Input | Expected |
|----|-----------|-------|----------|
| S1 | `Open` creates DB | path to new file | DB created, schema applied |
| S2 | `Open` WAL mode | — | `journal_mode=WAL` |
| S3 | `InsertWakeEvent` | id, source, reason | row inserted |
| S4 | `HasUnconsumedWakeEvents` empty | no events | false |
| S5 | `HasUnconsumedWakeEvents` with events | 1 unconsumed | true |
| S6 | `ConsumeWakeEvents` | 2 unconsumed | count=2, consumed=1 |
| S7 | `ConsumeWakeEvents` after consume | — | count=0 |
| S8 | `SetKV` then `GetKV` | key="x", value="y" | GetKV returns "y", true |
| S9 | `GetKV` missing key | key="nonexistent" | "", false, nil |
| S10 | `Close` | — | no error, DB unusable after |
| S11 | Schema tables exist | — | `turns`, `kv`, `wake_events`, `policy_decisions`, etc. |

---

## 7. Heartbeat (`internal/heartbeat`)

| ID | Test Case | Input | Expected |
|----|-----------|-------|----------|
| H1 | `DefaultTasks` | — | 3 tasks (heartbeat_ping, check_credits, check_usdc_balance) |
| H2 | `Daemon.Start` then `Stop` | tick 100ms | no panic, goroutine exits |
| H3 | Task runs on tick | — | task Run called |
| H4 | Context cancel stops daemon | ctx.Done() | daemon stops |

---

## 8. Agent Loop (`internal/agent`)

| ID | Test Case | Input | Expected |
|----|-----------|-------|----------|
| A1 | `RunOneTurn` | state=waking | returns state, nil |
| A2 | `ShouldSleep` idleTurns=2 | — | false |
| A3 | `ShouldSleep` idleTurns=3 | — | true |

---

## 9. CLI Commands (`cmd/`)

| ID | Test Case | Command | Expected |
|----|-----------|---------|----------|
| CLI1 | `--help` | `moneyclaw --help` | usage, subcommands listed |
| CLI2 | `--version` | `moneyclaw -v` | version string |
| CLI3 | `init` | `moneyclaw init` | `~/.automaton` created |
| CLI4 | `init` idempotent | run twice | no error |
| CLI5 | `setup` no config | first run | prompts, config saved |
| CLI6 | `setup` existing config | config exists | "Use configure to edit" |
| CLI7 | `status` no config | no config | "Run setup first" |
| CLI8 | `status` with config | config exists | Config path, DB path, Name, Conway |
| CLI9 | `run` no config | no config | error "run setup first" |
| CLI10 | `run` with config | config exists | bootstrap starts, heartbeat runs (Ctrl+C to stop) |

---

## 10. Integration Flows

| ID | Flow | Steps | Expected |
|----|------|-------|----------|
| I1 | Fresh install | init → setup (stdin) → status | config saved, status shows values |
| I2 | Run bootstrap | run (with config) | DB opened, policy engine created, heartbeat started |
| I3 | Wake events | InsertWakeEvent → HasUnconsumed → ConsumeWakeEvents | event consumed, count correct |
| I4 | Config round-trip | Save → Load | config equal |
| I5 | Policy + DB | Evaluate tool → Insert policy_decisions (future) | audit trail |

---

## 11. Security

| ID | Test Case | Input | Expected |
|----|-----------|-------|----------|
| SEC1 | Path protection constitution | tool path contains "constitution" | deny |
| SEC2 | Path protection wallet | tool path contains "wallet" | deny |
| SEC3 | Path protection state.db | tool path contains "state.db" | deny |
| SEC4 | Path protection api-key read | tool path contains "api-key" | deny |
| SEC5 | Authority external dangerous | source=external, risk=dangerous | deny |
| SEC6 | Authority self dangerous | source=self, risk=dangerous | allow |

---

## 12. Execution Matrix

| Target | Command |
|--------|---------|
| Unit tests | `go test ./internal/... -v` |
| All tests | `make test` |
| Coverage | `make test-coverage` |
| CLI smoke | `./bin/moneyclaw --help && ./bin/moneyclaw init && ./bin/moneyclaw status` |
| E2E (manual) | `AUTOMATON_DIR=/tmp/moneyclaw-test ./bin/moneyclaw init && echo "agent\nprompt\n0x0\n\n" \| ./bin/moneyclaw setup && ./bin/moneyclaw run` (Ctrl+C) |

---

## 13. Test File Mapping

| Package | Test File | IDs Covered |
|---------|-----------|-------------|
| `internal/config` | `config_test.go` | C1–C11 |
| `internal/types` | `types_test.go` | T1–T4 |
| `internal/conway` | `credits_test.go` | CR1–CR8 |
| `internal/agent` | `policy_test.go`, `policy_rules_test.go` | P1–P16 |
| `internal/state` | `database_test.go` | S1–S11 |
| `internal/heartbeat` | `daemon_test.go`, `tasks_test.go` | H1–H4 |
| `internal/agent` | `loop_test.go` | A1–A3 |

---

## 14. Acceptance Criteria

- [x] All unit tests pass: `make test`
- [ ] Coverage ≥ 70% for `internal/` (optional)
- [ ] No race conditions: `go test -race ./...`
- [ ] CLI smoke (init, setup, status) succeeds
- [ ] Run bootstrap completes without panic (manual stop)
- [ ] Policy engine denies protected paths and external dangerous tools
