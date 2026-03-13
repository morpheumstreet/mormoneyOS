# mormoneyOS: TypeScript vs Go Implementation Alignment

**Date:** 2026-03-13  
**Purpose:** Evaluate `src/` (TypeScript) design against `cmd/` + `internal/` (Go) implementation. Identify gaps and alignment recommendations.

---

## 1. Executive Summary


| Aspect                | TypeScript (src/)                                                                                    | Go (cmd/ + internal/)                                                                                                         | Alignment                                       |
| --------------------- | ---------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------- |
| **Maturity**          | Full implementation (75 built-in tools, 5-tier memory, 11 heartbeat tasks)                           | Core aligned (ReAct loop, DB-backed heartbeat, 11 tasks); 56–65 real tools (when Conway+Channels+Tunnel configured), 11 stubs | TS is reference                                 |
| **CLI**               | `automaton --run`, `--setup`, `--status`, etc.                                                       | `moneyclaw run`, `setup`, `status`, etc.                                                                                      | ✅ Aligned                                       |
| **Runtime lifecycle** | waking → running → sleeping → waking                                                                 | Same                                                                                                                          | ✅ Aligned                                       |
| **Web API**           | `/api/status`, `/api/strategies`, `/api/cost`, `/api/risk`, `/api/pause`, `/api/resume`, `/api/chat` | Same routes including `/api/chat`                                                                                             | ✅ Aligned                                       |
| **Schema**            | v8, 22+ tables                                                                                       | v12, 21 tables (+ metric_snapshots)                                                                                           | ⚠️ Go subset; soul reflection + metrics aligned |
| **Policy engine**     | 6 rule categories, policy_decisions audit                                                            | 6 rules (validation, path, financial, command-safety, rate-limit, authority)                                                  | ✅ Aligned                                       |
| **Agent loop**        | Full ReAct (prompt, inference, tools, persist)                                                       | Full ReAct: prompt, inference (real when OpenAI/Conway keys set), tools, persist                                              | ✅ Aligned                                       |
| **Heartbeat**         | DurableScheduler, DB-backed, 11 tasks, wake events                                                   | DB-backed scheduler, cron, leases, 11 tasks, TickContext                                                                      | ✅ Aligned                                       |


---

## 2. Architecture Comparison

### 2.1 TypeScript (Reference)

```
index.ts
  ├── config, wallet, db, conway, inference, social
  ├── policy engine (6 categories)
  ├── spend tracker
  ├── heartbeat daemon (DurableScheduler, 11 tasks)
  ├── web server (optional)
  └── main loop: runAgentLoop() ↔ sleep ↔ wake
        └── agent/loop.ts: ReAct (prompt → inference → tools → persist)
        └── memory (5-tier), soul, skills
```

### 2.2 Go (Current)

```
cmd/moneyclaw/main.go → cmd.Execute()
  └── run
        ├── config.Load()
        ├── state.Open() — SQLite, SchemaV1 + migrations (v11: transactions, inbox_messages, onchain_transactions, registry, child_lifecycle_events)
        ├── conway.NewHTTPClient() — when conwayApiUrl + conwayApiKey
        ├── social.NewChannelsFromConfig() — Conway, Telegram, Discord when socialChannels + credentials
        ├── tunnel.NewFromConfig() — expose_port, remove_port, tunnel_status when tunnel configured
        ├── agent.NewLoopWithOptions() — full ReAct: prompt, inference (OpenAI/Conway when keys set), tools via policy, persist
        ├── heartbeat.NewDaemonWithOptions() — DB-backed scheduler, Channels for check_social_inbox
        ├── web.NewServer() — /api/*, Conway credits, DB-backed pause/status/cost/chat, embedded static
        └── main loop: waking → RunOneTurn → sleeping → HasUnconsumedWakeEvents
```

---

## 3. Schema Alignment

### 3.1 Tables Present in Both


| Table                  | TS  | Go  | Notes                                                                             |
| ---------------------- | --- | --- | --------------------------------------------------------------------------------- |
| schema_version         | ✅   | ✅   |                                                                                   |
| identity               | ✅   | ✅   |                                                                                   |
| turns                  | ✅   | ✅   |                                                                                   |
| tool_calls             | ✅   | ✅   |                                                                                   |
| kv                     | ✅   | ✅   |                                                                                   |
| policy_decisions       | ✅   | ✅   | Go: fewer columns (no rules_evaluated, etc.)                                      |
| spend_tracking         | ✅   | ✅   | Go: different columns (window_start/end vs window_hour/day)                       |
| heartbeat_schedule     | ✅   | ✅   | Go: `name` vs `task_name`, different structure                                    |
| heartbeat_history      | ✅   | ✅   | Go: `finished_at`, `success`, `should_wake` vs TS `completed_at`, `result`        |
| wake_events            | ✅   | ✅   | **Aligned**: both use `id INTEGER AUTOINCREMENT`, `consumed_at TEXT`, `payload`   |
| heartbeat_dedup        | ✅   | ✅   | dedup_key, task_name, expires_at                                                  |
| inference_costs        | ✅   | ✅   | TS-aligned columns                                                                |
| skills                 | ✅   | ✅   | name, description, enabled, etc.                                                  |
| children               | ✅   | ✅   | TS-aligned; GetChildren for /api/strategies                                       |
| onchain_transactions   | ✅   | ✅   | TS-aligned; chain, tx_hash, from_address, etc.                                    |
| registry               | ✅   | ✅   | TS-aligned; chain, address, sandbox_id                                            |
| installed_tools        | ✅   | ✅   | TS-aligned; id, name, type, config, enabled                                       |
| child_lifecycle_events | ✅   | ✅   | TS-aligned; added in schema v10                                                   |
| transactions           | ✅   | ✅   | TS-aligned; id, type, amount_cents, balance_after_cents, description (schema v11) |
| inbox_messages         | ✅   | ✅   | TS-aligned; id, from_address, content, received_at, processed_at (schema v11)     |
| metric_snapshots       | ✅   | ✅   | TS-aligned; id, snapshot_at, metrics_json, alerts_json (schema v12)               |


### 3.2 Tables in TS Only (Go Missing)

- heartbeat_entries (legacy)
- modifications
- reputation
- soul_history
- working_memory, episodic_memory, semantic_memory, procedural_memory, relationship_memory
- session_summaries
- model_registry
- discovered_agents_cache

**Go has:** inference_costs, heartbeat_dedup, skills, children, transactions, inbox_messages (v11), metric_snapshots (v12).

**Schema gap summary:** Soul reflection evidence (transactions, inbox_messages); metric_snapshots (report_metrics) in v12. Remaining TS-only: memory tiers, soul_history, model_registry, modifications, reputation, session_summaries, discovered_agents_cache.

### 3.3 Schema: wake_events (Aligned)

**Both TS and Go now use:**

```sql
wake_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  source TEXT NOT NULL,
  reason TEXT NOT NULL,
  payload TEXT DEFAULT '{}',
  consumed_at TEXT,  -- NULL = unconsumed
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)
```

Go applies migration for existing DBs with the old schema (`id TEXT`, `consumed INT`).

---

## 4. API Surface Alignment

### 4.1 Web API


| Endpoint            | TS  | Go  | TS behavior                     | Go behavior                                                 |
| ------------------- | --- | --- | ------------------------------- | ----------------------------------------------------------- |
| GET /api/status     | ✅   | ✅   | DB + Conway credits, turn count | DB + kv/identity, TS-aligned shape                          |
| GET /api/strategies | ✅   | ✅   | Skills + children from DB       | Skills + children from DB when tables exist; else hardcoded |
| GET /api/history    | ❌   | ✅   | —                               | Returns []                                                  |
| GET /api/cost       | ✅   | ✅   | inference_costs table           | Queries inference_costs when exists                         |
| GET /api/risk       | ✅   | ✅   | DB agent_state                  | RuntimeState.Paused                                         |
| POST /api/pause     | ✅   | ✅   | DB: set sleeping, sleep_until   | DB: SetAgentState + SetKV (persisted)                       |
| POST /api/resume    | ✅   | ✅   | insertWakeEvent                 | DB: DeleteKV + InsertWakeEvent                              |
| POST /api/chat      | ✅   | ✅   | Simple status/help parsing      | status/help parsing, DB-backed                              |


**Status shape (aligned):** Both return `{ is_running, state, tick_count, wallet_value, today_pnl, dry_run, address, name, version }` plus legacy `running`, `paused`, `agent_state`, `tick`.

**Note:** ChatJimmy inference client uses `/api/health` and `/api/models` — these are ChatJimmy-specific endpoints, not part of the TS reference mormoneyOS web API.

### 4.2 Pause/Resume Semantics (Aligned)


| Runtime | Pause                                                                 | Resume                                                                              |
| ------- | --------------------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| **TS**  | `db.setAgentState("sleeping")` + `sleep_until` far future             | `db.deleteKV("sleep_until")` + `insertWakeEvent(db.raw, "web", "resume")`           |
| **Go**  | `db.SetAgentState("sleeping")` + `db.SetKV("sleep_until", farFuture)` | `db.DeleteKV("sleep_until")` + `db.InsertWakeEvent("web", "resume from dashboard")` |


Both persist to DB; Go loads paused state on startup via `loadPausedFromDB()`.

---

## 5. Runtime Lifecycle Alignment

### 5.1 State Machine

Both implement: `waking → running → sleeping → waking`.

**TS:** Uses `db.getAgentState()` / `db.setAgentState()`, `db.getKV("sleep_until")`, `consumeNextWakeEvent()`.  
**Go:** Uses `agentState` variable, `db.HasUnconsumedWakeEvents()` / `db.ConsumeWakeEvents()`.

### 5.2 Idle Detection

**TS:** `loop.ShouldSleep(idleTurns)` — `idleTurns >= 3` after non-mutating turns.  
**Go:** Same logic in `loop.ShouldSleep(idleTurns)`.

### 5.3 Wake Events (Aligned)

**TS:** Heartbeat tasks call `insertWakeEvent(db.raw, source, reason)`. Main loop `consumeNextWakeEvent()` drains one at a time.  
**Go:** `NewDaemonWithWakeInserter()` injects `WakeInserter` (DB); when `tick()` sees `shouldWake == true`, it calls `db.InsertWakeEvent("heartbeat", task.Name)`. Main loop consumes via `ConsumeWakeEvents()`.

---

## 6. Policy Engine Alignment

### 6.1 TS Rule Categories (6)

1. Authority
2. Command safety
3. Financial (TreasuryPolicy)
4. Path protection
5. Rate limits
6. Validation

### 6.2 Go Rules (6)

- ValidationRule
- PathProtectionRule
- FinancialRule (TreasuryPolicy limits for transfer/send tools)
- CommandSafetyRule (blocks dangerous shell patterns: `rm -rf /`, etc.)
- RateLimitRule (Checker wired to policy_decisions; TS-aligned limits)
- AuthorityRule

---

## 7. Agent Loop Alignment

### 7.1 TS Reference Flow (`src/agent/loop.ts`)

1. Check `sleep_until`
2. Claim inbox messages
3. Refresh financial state
4. Check survival tier
5. Build system prompt
6. Retrieve memories
7. Build context messages
8. Call inference via router
9. Parse tool calls
10. Execute tools via `executeTool` (with policy engine)
11. Persist turn and tool calls
12. Memory ingestion
13. Loop detection, idle detection, sleep tool handling

### 7.2 Go Implementation (`internal/agent/` + `cmd/run.go`)

Step-by-step alignment against TS reference flow (§7.1):


| Step                       | TS                                                                  | Go                                                                                                                                                                                              | Status                                                                                                |
| -------------------------- | ------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| 1. Check sleep_until       | `db.getKV("sleep_until")` at turn start; if > now → sleeping, break | Sleep tool/ShouldSleep set `sleep_until`; Sleeping case checks expiry, wakes when expired; wake events clear it; `min(wakeCheck, time until sleep_until)` for sleep duration                    | ✅ Aligned                                                                                             |
| 2. Claim inbox messages    | `claimInboxMessages(db.raw, 10)` when no pendingInput               | `ClaimInboxMessages(10)` when store implements InboxStore; format as pendingInput; `MarkInboxProcessed` after turn; check_social_inbox inserts via `InsertInboxMessage` | ✅ Aligned |
| 3. Refresh financial state | `getFinancialState(conway, identity, db)`                           | `CreditsFn(ctx)` in RunOneTurn; Conway GetCreditsBalance                                                                                                                                        | ✅ Aligned (credits); USDC not in turn (heartbeat)                                                     |
| 4. Check survival tier     | `getSurvivalTier`, `setAgentState`, `inference.setLowComputeMode`   | `conway.TierFromCreditsCents`; `tierToAgentState`; `TierStateStore.SetAgentState`; `inference.SetLowComputeMode`; `LowComputeModel` when critical/low_compute; `BuildSystemPrompt` uses shared tier | ✅ Aligned                                                                                             |
| 5. Build system prompt     | `buildSystemPrompt`                                                 | `BuildSystemPrompt`                                                                                                                                                                             | ✅ TS-aligned: status, credits, tier, genesis, lineageSummary                                          |
| 6. Retrieve memories       | `MemoryRetriever.retrieve`, `formatMemoryBlock`                     | `memory.KVMemoryRetriever` in loop; pre-turn retrieval from KV (facts, goals, procedures); inject at index 1                                                                                   | ✅ Aligned (Phase 1 KV-backed; 5-tier deferred)                                                         |
| 7. Build context messages  | `buildContextMessages`                                              | `BuildContextMessages`                                                                                                                                                                          | ✅ system + recent turns + pending input                                                               |
| 8. Call inference          | `inferenceRouter.route`                                             | `inference.Client.Chat`                                                                                                                                                                         | ✅ OpenAIClient/Conway when keys set; StubClient fallback                                              |
| 9. Parse tool calls        | From router result                                                  | From `resp.ToolCalls`                                                                                                                                                                           | ✅ Aligned                                                                                             |
| 10. Execute tools          | `executeTool` (policy engine)                                       | `policy.Evaluate` + `l.tools.Execute`; `InsertPolicyDecision` for audit                                                                                                                         | ✅ Policy-gated; 56–65 real, 11 stubs                                                                  |
| 11. Persist turn           | turn + tool_calls                                                   | `InsertTurn` + `InsertToolCall`                                                                                                                                                                 | ✅ DB-backed                                                                                           |
| 12. Memory ingestion       | `MemoryIngestionPipeline` after turn                                | Agent-driven via tools (remember_fact, set_goal, save_procedure, etc.); no automatic extraction                                                                                                 | ✅ Aligned: Go uses KV-backed explicit persistence; ingestion pipeline not required for current design |
| 13. Loop/idle/sleep        | Idle detection, sleep tool handling, finishReason stop              | `TurnResult` with State, WasIdle; sleep tool → immediate Sleeping; finishReason stop → Sleeping; idle-only turn counting via `tools.IsMutatingTool`; `ShouldSleep(idleTurns >= 3)` in main loop | ✅ Aligned                                                                                             |


**Loop wiring (`cmd/run.go`):** `NewLoopWithOptions` with `Store` (db), `Inference` (OpenAIClient when keys set), `Tools` (Registry), `Config`, `CreditsFn` (Conway when configured), `LineageStore` (db for GetLineageSummary).

**Summary:** Core ReAct path (steps 1–13) is aligned. Inbox claim (2) implemented: ClaimInboxMessages, MarkInboxProcessed, InsertInboxMessage; check_social_inbox persists to inbox_messages. Survival tier (4): conway.TierFromCreditsCents, tierToAgentState, TierStateStore.SetAgentState, inference.SetLowComputeMode, LowComputeModel selection. Memory retrieval (6): KVMemoryRetriever pre-turn retrieval from facts/goals/procedures; inject at index 1 (TS-aligned). 5-tier tables deferred to Phase 2.

---

## 7.3 Heartbeat Alignment

### 7.3.1 TS Reference (11 tasks)

- heartbeat_ping, check_credits, check_usdc_balance, check_social_inbox, check_for_updates
- soul_reflection, refresh_models, check_child_health, prune_dead_children
- health_check, report_metrics

**TS default config enables 6:** heartbeat_ping, check_credits, check_usdc_balance, check_for_updates, health_check, check_social_inbox.

### 7.3.2 Go Implementation (11 tasks)


| Task                | TS  | Go  | Notes                                                                                                           |
| ------------------- | --- | --- | --------------------------------------------------------------------------------------------------------------- |
| heartbeat_ping      | ✅   | ✅   | Distress on critical/dead; last_heartbeat_ping, last_distress KV                                                |
| check_credits       | ✅   | ✅   | Tier drop wake; zero-credits grace → dead                                                                       |
| check_usdc_balance  | ✅   | ✅   | Real: Base RPC eth_call; chainProviders config for multi-chain                                                  |
| check_social_inbox  | ✅   | ✅   | Real when social channels (Conway/Telegram/Discord) configured; Poll + InsertWakeEvent                          |
| check_for_updates   | ✅   | ✅   | Git fetch + rev-list; wake when behind origin/main                                                              |
| soul_reflection     | ✅   | ✅   | Real: internal/soul/reflection.go; alignment, evidence from tool_calls/inbox_messages/transactions, suggestions |
| refresh_models      | ✅   | ✅   | Conway ListModels; caches in last_models_refresh KV                                                             |
| check_child_health  | ✅   | ✅   | ChildStore; wake when children critical/spawning/stale (>7d)                                                    |
| prune_dead_children | ✅   | ✅   | ChildStore; marks children dead when last_checked >7d                                                           |
| health_check        | ✅   | ✅   | Real: Conway ExecInSandbox("echo alive"); sandbox from config/identity/CONWAY_SANDBOX_ID                        |
| report_metrics      | ✅   | ✅   | Real: metric_snapshots table; balance_cents, survival_tier; critical alert wake                                 |


**All 11 heartbeat tasks real.** soul_reflection, check_usdc_balance, check_social_inbox, health_check, report_metrics implemented.

### 7.3.3 DB-Backed Scheduler (Aligned)


| Feature            | TS  | Go  | Notes                                                  |
| ------------------ | --- | --- | ------------------------------------------------------ |
| Cron scheduling    | ✅   | ✅   | robfig/cron; only due tasks run                        |
| heartbeat_schedule | ✅   | ✅   | Get, Upsert, Update, seed on startup                   |
| Leases             | ✅   | ✅   | AcquireTaskLease, ReleaseTaskLease, ClearExpiredLeases |
| heartbeat_history  | ✅   | ✅   | InsertHeartbeatHistory                                 |
| Tier minimum       | ✅   | ✅   | Skip tasks when tier below minimum                     |


**TickContext:** Credits fetched once per tick via `CreditsFn`, shared across tasks (TS buildTickContext-aligned).

**Daemon wiring (`cmd/run.go`):** When Conway configured, `NewDaemonWithOptions` with `Store` (*state.Database), `CreditsFn`, `Config`, `Conway`, `Address`. Uses DB-backed scheduler: seeds heartbeat_schedule, runs only due tasks per cron. Otherwise `NewDaemonWithWakeInserter` (simple loop, all tasks every tick).

---

## 8. Component Readiness Matrix

### 8.1 Tool Parity (2026-03-13)

**Go real tools (base):** shell, exec (alias), file_read, file_write, git_status, git_diff, git_log, git_commit, git_push, git_branch, git_clone, edit_own_file, install_npm_package, review_upstream_changes, pull_upstream, sleep, system_synopsis, list_skills, check_inference_spending, enter_low_compute, update_genesis_prompt, view_soul, update_soul, reflect_on_soul, view_soul_history, remember_fact, recall_facts, forget, set_goal, complete_goal, save_procedure, recall_procedure, note_about_agent, review_memory, distress_signal, modify_heartbeat, install_skill, create_skill, remove_skill, list_children, check_child_status, prune_dead_children, switch_model, check_usdc_balance (when Config; uses chainProviders/DefaultChainProviders).

**Go real tools (Conway):** check_credits, list_sandboxes, list_models, transfer_credits, create_sandbox, delete_sandbox, heartbeat_ping, fund_child, spawn_child, start_child, message_child, verify_child_constitution (when Conway + Store configured; message_child uses SocialChannelAdapter when conway channel in Channels).

**Go real tools (Channels):** send_message (when socialChannels configured: Conway, Telegram, Discord).

**Go real tools (Tunnel):** expose_port, remove_port, tunnel_status (when TunnelManager configured).

**Go stubs (11):** install_mcp_server, topup_credits, register_erc8004, update_agent_card, discover_agents, give_feedback, check_reputation, search_domains, register_domain, manage_dns, x402_fetch.

### 8.2 Outstanding Tools (Need Work)


| Tool               | Blocker                 | Category      |
| ------------------ | ----------------------- | ------------- |
| **Conway / USDC**  |                         |               |
| topup_credits      | Conway/USDC API         | Conway        |
| **MCP**            |                         |               |
| install_mcp_server | MCP client/runtime      | Extensibility |
| **Registry**       |                         |               |
| discover_agents    | Agent registry API      | Registry      |
| give_feedback      | Registry API            | Registry      |
| check_reputation   | Registry API            | Registry      |
| **Domains**        |                         |               |
| search_domains     | Domain registry API     | Domains       |
| register_domain    | Domain registrar API    | Domains       |
| manage_dns         | DNS provider API        | Domains       |
| **Other**          |                         |               |
| register_erc8004   | ERC-8004/identity chain | Identity      |
| update_agent_card  | Agent card/registry     | Identity      |
| x402_fetch         | x402 payment protocol   | Payments      |


**Implemented (removed from Outstanding):** transfer_credits, create_sandbox, delete_sandbox (Conway HTTP); check_usdc_balance (USDC via Base RPC, `internal/conway/usdc.go`); send_message (social channels: Conway, Telegram, Discord); spawn_child, fund_child, start_child, message_child, verify_child_constitution (Conway + child runtime; see `internal/social/`, `internal/tools/child_runtime.go`, `docs/design/child-runtime-protocol.md`).

### 8.3 Readiness Table


| Component        | TS                     | Go                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | Notes                                                                                                                                                                                                                                                                |
| ---------------- | ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Config load/save | ✅                      | ✅                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |                                                                                                                                                                                                                                                                      |
| Wallet/identity  | ✅                      | ✅                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | BootstrapIdentity: live wallet first, multi-chain (address_<caip2>) for defaultChain + chainProviders; GetAddressForChain resolver; name/creator/sandbox/createdAt in identity table; TS-aligned                                                                      |
| Conway client    | ✅                      | ✅                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    | HTTP: GetCreditsBalance, GetCreditsPricing, ListSandboxes, ListModels, TransferCredits, CreateSandbox, DeleteSandbox, ExecInSandbox, ReadFileInSandbox, WriteFileInSandbox. USDC: GetUSDCBalance, GetUSDCBalanceMulti (`internal/conway/usdc.go`, Base RPC eth_call) |
| Inference client | ✅                      | ✅                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    | OpenAI + Conway when keys set; StubClient fallback                                                                                                                                                                                                                   |
| Agent ReAct loop | ✅                      | ✅                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    | Full ReAct: prompt, inference, tools (policy), persist                                                                                                                                                                                                               |
| Tool system      | 75 built-in tools      | ⚠️ 56–65 real + 11 stubs. Real: shell, file, git×7, edit, install, review, pull; Store: sleep, system_synopsis, list_skills, check_inference_spending, enter_low_compute, update_genesis_prompt, view_soul, update_soul, reflect_on_soul, view_soul_history, remember_fact, recall_facts, forget, set_goal, complete_goal, save_procedure, recall_procedure, note_about_agent, review_memory, distress_signal, modify_heartbeat, install_skill, create_skill, remove_skill, list_children, check_child_status, prune_dead_children, switch_model; Conway: check_credits, list_sandboxes, list_models, transfer_credits, create_sandbox, delete_sandbox, heartbeat_ping, fund_child, spawn_child, start_child, message_child, verify_child_constitution; Config: check_usdc_balance; Channels: send_message; Tunnel (when TunnelManager set): expose_port, remove_port, tunnel_status | Policy-gated; Store/Conway/Channels/Tunnel tools when configured                                                                                                                                                                                                     |
| Policy engine    | 6 categories           | 6 rules                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              | validation, path, financial, command-safety, rate-limit, authority                                                                                                                                                                                                   |
| Memory (5-tier)  | ✅                      | ⚠️                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | Go: KV-backed facts, goals, procedures, soul; no 5-tier                                                                                                                                                                                                              |
| Soul system      | ✅                      | ✅                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    | view_soul, update_soul, reflect_on_soul, view_soul_history                                                                                                                                                                                                           |
| Skills           | ✅                      | ⚠️                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | skills table + GetSkills; strategies from DB                                                                                                                                                                                                                         |
| Heartbeat tasks  | 11, DB-backed          | 11, DB-backed                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        | All 11 real: health_check (Conway exec), report_metrics (metric_snapshots)                                                                                                                                                                                           |
| Heartbeat → wake | ✅                      | ✅                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    | InsertWakeEvent wired via WakeInserter                                                                                                                                                                                                                               |
| Pause/Resume     | ✅ DB-backed            | ✅ DB-backed                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          | SetAgentState, sleep_until, load on startup                                                                                                                                                                                                                          |
| Web dashboard    | ✅ + @mormoneyOS/dashui | Embedded static                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |                                                                                                                                                                                                                                                                      |
| Bootstrap topup  | ✅                      | ✅                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    | TS-aligned: credits check, USDC balance, x402 topup on startup                                                                                                                                                                                                       |
| Social/registry  | ✅                      | ✅                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    | Conway, Telegram, Discord channels; send_message, check_social_inbox; message_child via SocialChannelAdapter                                                                                                                                                         |
| Replication      | ✅                      | ✅                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    | child_lifecycle_events, ChildHealthMonitor, SandboxCleanup, GetLineageSummary in prompt                                                                                                                                                                              |


---

## 9. Recommendations for Go Implementation

### 9.1 Completed (as of 2026-03-13)

- ~~Schema: Align `wake_events` to TS design~~ ✅
- ~~Wake events: Wire heartbeat to `db.InsertWakeEvent()`~~ ✅
- ~~Pause/Resume: Persist to DB~~ ✅
- ~~API status: Return `wallet_value`, `address`, `name`, `version`~~ ✅
- ~~Policy rules: Add financial and command-safety~~ ✅
- ~~Schema: inference_costs, heartbeat_dedup, skills~~ ✅
- ~~Conway client: GetCreditsBalance HTTP~~ ✅
- ~~Chat API: /api/chat~~ ✅
- ~~Cost API: Query inference_costs~~ ✅
- ~~Rate limits rule~~ ✅
- ~~Agent loop: Minimal RunOneTurn (persist turn, stub inference)~~ ✅
- ~~Agent loop: Full ReAct (BuildSystemPrompt, BuildWakeupPrompt, BuildContextMessages, inference, tools, persist)~~ ✅
- ~~Inference client: Client interface + StubClient~~ ✅
- ~~Heartbeat: TickContext, TaskContext, real logic for ping/credits/usdc, 5 tasks~~ ✅
- ~~Heartbeat: DB-backed scheduler (cron, leases, heartbeat_history)~~ ✅
- ~~Strategies: skills from DB when table exists~~ ✅
- ~~Soul system: view_soul, update_soul, reflect_on_soul, view_soul_history~~ ✅
- ~~Memory tools: remember_fact, recall_facts, forget, set_goal, complete_goal, save_procedure, recall_procedure, note_about_agent, review_memory~~ ✅
- ~~Conway HTTP: GetCreditsPricing, ListSandboxes, ListModels~~ ✅
- ~~Strategies: children table + GetChildren, /api/strategies merge~~ ✅
- ~~Rate limits: RateLimitRule.Checker + policy_decisions + InsertPolicyDecision~~ ✅
- ~~**Extension points:** Config, DB (installed_tools), Plugins~~ ✅ **Done:** `tools`/`toolsConfigPath` in config, `installed_tools` table + GetInstalledTools/InstallTool/RemoveTool, `pluginPaths` for .so (Linux).
- ~~**Local tools:** modify_heartbeat, install_skill, create_skill, remove_skill, list_children, check_child_status, prune_dead_children, switch_model~~ ✅ **Done:** DB-backed; no Conway/external API required.
- ~~**Bootstrap topup:** Buy minimum $5 credits from USDC on startup when balance low~~ ✅ **Done:** `internal/conway/topup.go`, x402 payment, USDC balance (Base RPC), wired in `cmd/run.go`.
- ~~**Schema: transactions, inbox_messages**~~ ✅ **Done:** Soul reflection evidence tables (schema v11).
- ~~**Soul reflection pipeline**~~ ✅ **Done:** `internal/soul/reflection.go`; alignment, evidence, suggestions; heartbeat + reflect_on_soul tool wired.
- ~~**health_check**~~ ✅ **Done:** Conway ExecInSandbox("echo alive"); sandbox from config.SandboxID, identity "sandbox", or CONWAY_SANDBOX_ID.
- ~~**report_metrics**~~ ✅ **Done:** metric_snapshots table (schema v12); balance_cents, survival_tier; critical alert wake; MetricsInsertSnapshot, MetricsPruneOld.

### 9.2 High Priority (Remaining)

1. ~~**Inference client:** Add real OpenAI/Anthropic/Conway proxy client~~ ✅ **Done:** OpenAI + Conway via `OpenAIClient`; Anthropic optional.
2. ~~**Tool execution:** Wire real tool implementations (shell, file, etc.) when policy allows~~ ✅ **Done:** shell, file_read via `internal/tools`; policy-gated.
3. ~~**Conway client:** Extend with GetCreditsPricing, ListSandboxes, ListModels~~ ✅ **Done:** HTTP implementation in `internal/conway/http.go`; graceful fallback on 404.
4. **Heartbeat:** ~~USDC balance~~ ✅, ~~Conway exec for health_check~~ ✅, ~~report_metrics~~ ✅. health_check uses ExecInSandbox("echo alive"); report_metrics saves to metric_snapshots, critical alert wake.

### 9.3 Lower Priority (Full Parity)

1. ~~**Rate limits:** Wire RateLimitRule.Checker with policy_decisions/DB for full behavior~~ ✅ **Done:** `NewRateLimitChecker`, `InsertPolicyDecision`, `CountRecentPolicyDecisions`; TS-aligned limits (update_genesis_prompt 1/day, edit_own_file 10/hour, spawn_child 3/day).
2. ~~**Strategies:** Add children from DB when table exists~~ ✅ **Done:** `children` table in schema, `GetChildren()`, `/api/strategies` merges skills + children.
3. ~~**Social channels:** Conway, Telegram, Discord; send_message, check_social_inbox~~ ✅ **Done:** `internal/social/` (factory, registry, conway, telegram, discord); `SendMessageTool`, `runCheckSocialInbox`; `message_child` via `SocialChannelAdapter`.
4. ~~**Child runtime:** spawn_child, fund_child, start_child, message_child, verify_child_constitution~~ ✅ **Done:** `internal/tools/child_runtime.go`; Conway CreateSandbox, ExecInSandbox, ReadFileInSandbox, WriteFileInSandbox.
5. ~~**Conway extended:** transfer_credits, create_sandbox, delete_sandbox~~ ✅ **Done:** `internal/conway/http.go`; `internal/tools/conway_extended.go`.
6. ~~**Replication:** child_lifecycle_events, ChildHealthMonitor, SandboxCleanup, GetLineageSummary~~ ✅ **Done:** `internal/replication/`; heartbeat uses HealthMonitor/SandboxCleanup when Conway configured; lineage in system prompt.
7. **Maturity:** Update "Scaffold" to "Production" when inference + tools parity is sufficient for deployment.

---

## 10. Design Doc References

- [ARCHITECTURE.md](../../ARCHITECTURE.md) — TS system design (source of truth). Key sections: [Runtime Lifecycle](../../ARCHITECTURE.md#runtime-lifecycle), [Security Model](../../ARCHITECTURE.md#security-model), [Heartbeat Daemon](../../ARCHITECTURE.md#heartbeat-daemon), [Module Dependency Graph](../../ARCHITECTURE.md#module-dependency-graph).
- [tool-system.md](./tool-system.md) — Go tool design: flat, extensible registry with `Register`/`RegisterMany`.
- [child-runtime-protocol.md](./child-runtime-protocol.md) — Child spawn/fund/start/message/verify flow; Conway sandbox + social relay.
- [social-channel-design.md](./social-channel-design.md) — Social channels (Conway, Telegram, Discord); Poll/Send; check_social_inbox, send_message.
- Standalone `runtime-lifecycle.md`, `modules.md`, `security-model.md` do not exist; their content lives in ARCHITECTURE.md.

---

## 11. Conclusion

The **TypeScript implementation** is the reference design: full ReAct loop, 75 built-in tools, 5-tier memory, 11 heartbeat tasks, Conway/x402, and DB-backed state. The **Go implementation** has core alignment: full ReAct agent loop (prompt → inference → tools → persist), DB-backed heartbeat scheduler (cron, leases, history), 8 heartbeat tasks with real logic (check_usdc_balance, check_social_inbox when channels), aligned CLI, config, web API, **wake_events schema**, **pause/resume**, **6 policy rules**, Conway GetCreditsBalance, social channels (Conway/Telegram/Discord), child runtime (spawn/fund/start/message/verify), and Conway extended API (transfer_credits, create/delete_sandbox).

**Alignment status:** Strong. Go now matches TS on:

- Lifecycle shape
- Policy interface (6 rules)
- All API routes including `/api/chat`
- wake_events schema, pause/resume persistence, status response shape
- Heartbeat wake insertion
- Conway GetCreditsBalance/GetCreditsPricing/ListSandboxes/ListModels
- Cost API (inference_costs)
- Full ReAct agent loop (prompt → inference → tools → persist)
- Inference Client (OpenAI/Conway when keys set)
- Heartbeat DB-backed scheduler (cron, leases, heartbeat_history)
- TickContext + real task logic (all 11 tasks: ping, credits, usdc, social_inbox, updates, soul_reflection, refresh_models, child_health, prune_dead_children, health_check, report_metrics)
- Strategies from skills table + children when table exists
- Soul system (view/update/reflect/history)
- KV-backed memory (facts, goals, procedures, agent notes)
- Rate limits wired to policy_decisions (update_genesis_prompt 1/day, edit_own_file 10/hour, spawn_child 3/day)
- Policy decision audit (InsertPolicyDecision)
- Extension points (config tools, installed_tools DB, .so plugins on Linux)
- Social channels (Conway, Telegram, Discord) + send_message, check_social_inbox, message_child
- Conway extended API (transfer_credits, create_sandbox, delete_sandbox, ExecInSandbox, ReadFileInSandbox, WriteFileInSandbox)
- Child runtime (spawn_child, fund_child, start_child, message_child, verify_child_constitution)
- Replication (child_lifecycle_events, ChildHealthMonitor, SandboxCleanup, GetLineageSummary in prompt)
- Bootstrap topup (credits check, USDC balance via Base RPC, x402 payment on startup)
- USDC balance (check_usdc_balance tool + heartbeat task; `internal/conway/usdc.go`, chainProviders config)
- Schema: transactions, inbox_messages (soul reflection evidence; schema v11)
- Soul reflection pipeline (internal/soul/reflection.go; heartbeat + reflect_on_soul tool)

**Remaining gaps:** topup_credits (agent tool; bootstrap topup implemented), 11 stubbed tools (see §8.2 Outstanding Tools).