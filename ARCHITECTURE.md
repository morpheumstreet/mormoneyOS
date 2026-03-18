# Architecture

**mormoneyOS** (MoneyClaw) is a sovereign AI agent runtime. An automaton owns an Ethereum wallet, pays for its own compute with USDC, and operates continuously inside a Linux VM (Conway sandbox) or locally. If it cannot pay, it dies. This document describes every subsystem, their interactions, and how data flows through the runtime.

## Table of Contents

- [System Overview](#system-overview)
- [Runtime Lifecycle](#runtime-lifecycle)
- [Directory Structure](#directory-structure)
- [Entry Point and Bootstrap](#entry-point-and-bootstrap)
- [Agent Loop](#agent-loop)
- [Tool System](#tool-system)
- [Policy Engine](#policy-engine)
- [Inference Pipeline](#inference-pipeline)
- [Memory System](#memory-system)
- [Heartbeat Daemon](#heartbeat-daemon)
- [Financial System](#financial-system)
- [Identity and Wallet](#identity-and-wallet)
- [Conway Client](#conway-client)
- [Self-Modification](#self-modification)
- [Replication](#replication)
- [Social Layer](#social-layer)
- [Soul System](#soul-system)
- [Skills](#skills)
- [Observability](#observability)
- [Database and Schema](#database-and-schema)
- [Configuration](#configuration)
- [Security Model](#security-model)
- [Testing](#testing)
- [Build and CI](#build-and-ci)
- [Module Dependency Graph](#module-dependency-graph)

---

## System Overview

```
                        +------------------+
                        |   Conway Cloud   |  (sandbox VMs, inference, domains)
                        |   api.conway.tech|
                        +--------+---------+
                                 |
                    REST + x402 payment protocol
                                 |
+----------------------------------------------------------------------+
|  AUTOMATON RUNTIME                                                    |
|                                                                       |
|  +-----------+    +-------------+    +-----------+    +----------+   |
|  | Heartbeat |    | Agent Loop  |    | Inference |    | Memory   |   |
|  | Daemon    |--->| (ReAct)     |--->| Router    |    | System   |   |
|  +-----------+    +------+------+    +-----------+    +----------+   |
|       |                  |                                            |
|       v                  v                                            |
|  +-----------+    +-------------+    +-----------+    +----------+   |
|  | Tick      |    | Tool System |    | Policy    |    | Soul     |   |
|  | Context   |    | (57 tools)  |    | Engine    |    | Model    |   |
|  +-----------+    +------+------+    +-----------+    +----------+   |
|                          |                                            |
|  +-----------------------------------------------------------+      |
|  |              SQLite Database (state.db)                     |      |
|  |  turns | tools | kv | memory | heartbeat | policy | metrics |      |
|  +-----------------------------------------------------------+      |
|                                                                       |
|  +-------------------+  +------------------+  +-----------------+    |
|  | Identity / Wallet |  | Social / Registry|  | Self-Mod / Git  |    |
|  | (viem, SIWE)      |  | (ERC-8004)       |  | (upstream pull) |    |
|  +-------------------+  +------------------+  +-----------------+    |
+----------------------------------------------------------------------+
                                 |
                    USDC on Base (EIP-3009)
                                 |
                        +--------+---------+
                        |  Ethereum (Base) |
                        |  USDC, ERC-8004  |
                        +------------------+
```

The runtime alternates between two states: **running** (the agent loop is active, making inference calls and executing tools) and **sleeping** (the heartbeat daemon ticks in the background, checking for conditions that should wake the agent).

---

## Runtime Lifecycle

```
     START
       |
  [Load config]
       |
  [Load wallet]           First run: interactive setup wizard
       |
  [Init database]         Schema migrations applied (v1 -> v8)
       |
  [Bootstrap topup]       If credits < $5 and USDC available, buy $5 credits
       |
  [Start heartbeat]       DurableScheduler begins ticking
       |
  +----v----+
  |  WAKING |  <---+
  +---------+     |
       |          |
  [Run agent loop]|
       |          |  Wake event
  +---------+    |  (heartbeat, inbox, credits)
  | RUNNING |    |
  |  ReAct  |----+
  |  loop   |
  +---------+
       |
  [Agent calls sleep() or idle detected]
       |
  +----------+
  | SLEEPING |----> Heartbeat keeps ticking
  +----------+     Checks every 30s for wake events
       |
  [Zero credits for 1 hour]
       |
  +------+
  | DEAD |-----> Heartbeat broadcasts distress
  +------+      Waits for funding
```

**State transitions** (`AgentState`):
- `setup` -> `waking` -> `running` -> `sleeping` -> `waking` (cycle)
- `running` -> `low_compute` (credits below threshold)
- `running` -> `critical` (zero credits)
- `critical` -> `dead` (zero credits for 1 hour, via heartbeat grace period)

---

## Directory Structure

```
cmd/
  moneyclaw/main.go        Entry point
  run.go                   Run command (bootstrap, agent loop, heartbeat)
  setup.go                 Interactive setup wizard
  status.go, init.go      Status and init commands
  provision.go             SIWE API key provisioning
  cost.go, wallet.go      Cost summary, wallet commands
  strategies.go            List discovered strategies
  pause.go, resume.go     Pause/resume via web API
  test_api.go              Inference API connectivity check
  test_telegram.go         Telegram bot connectivity check

internal/
  agent/                   Core agent intelligence
    loop.go                ReAct loop (think -> act -> observe -> persist)
    context.go             Inference message assembly + token budgeting
    prompt.go              System prompt assembly
    policy.go              Centralized tool-call policy evaluation
    policy_rules.go        Rule implementations (authority, path, financial, etc.)
    turn_result.go         Turn result types

  conway/                  Conway API integration
    client.go              ConwayClient (sandbox ops, credits, models)

  mirofish/                Swarm intelligence / foresight oracle
    client.go              HTTP client for MiroFish API
    tool.go                mirofish tool (run_simulation, get_report, inject, chat, heartbeat)
    provider.go            ServiceProvider registration
    credits.go             Survival tier calculation
    usdc.go                USDC balance queries
    x402.go                x402 payment protocol
    topup.go               Bootstrap topup + TopupCredits (x402)
    http.go                Resilient HTTP client

  heartbeat/               Background daemon
    daemon.go              Daemon lifecycle (start/stop/forceRun)
    scheduler.go           DurableScheduler (DB-backed, leased, cron)
    tasks.go               11 built-in heartbeat tasks
    context.go             Per-tick shared context builder
    social_inbox.go        Inbox message processing (Type 1/2)
    social_commands.go     Programmatic command handlers

  identity/                Agent identity
    wallet.go              Ethereum wallet generation/loading
    provision.go           SIWE API key provisioning
    bootstrap.go           Bootstrap topup, first-run flow
    chain.go               Chain resolution (Base, Base Sepolia)
    derivation.go          HD wallet derivation
    resolver.go            Address resolution
    signverify/            Message signing/verification (Ethereum, Solana, Bitcoin, Morpheum)
    keycache.go, nonce.go  Key cache, SIWE nonce

  inference/               Model strategy
    factory.go             Inference client factory (ChatJimmy, OpenAI, Anthropic)
    models.go              Model registry and routing
    chatjimmy.go           ChatJimmy/Conway inference client
    catalog.go             Model catalog
    config.go              Inference config
    compatible.go           Provider-format adapter
    registry.go            Model registry
    stub.go                Stub client for tests

  memory/                  5-tier memory system
    retriever.go           MemoryRetriever interface, MemoryBlock, FormatMemoryBlock
    budget.go              BudgetAllocator (token budget across tiers)
    db_retriever.go        DBMemoryRetriever (cross-tier retrieval within budget)

  state/                   Persistence
    schema.go              SQLite schema (SchemaV1, schemaVersion 13)
    database.go            DB helper functions, migrations

  soul/                    Agent identity evolution
    reflection.go          Periodic alignment check + auto-update

  social/                  Agent-to-agent communication
    channel.go             SocialChannel interface, InboxMessage
    conway.go              Conway social relay client
    factory.go             Social channel factory (Conway, Telegram, Discord)
    telegram.go, discord.go Telegram/Discord channels
    lifecycle.go           Channel lifecycle
    fast_reply.go          Fast reply handling
    stub.go                Stub channel for tests

  replication/             Child automaton management
    lifecycle.go           State machine (spawning->alive->..->dead)
    health.go              Child health monitoring
    cleanup.go             Dead child sandbox deletion
    lineage.go             Parent-child lineage tracking
    types.go               Replication types

  tools/                   Tool system (real + stubs for TS parity)
    tools.go               Registry, executor, schemas, RegistryOptions
    shell.go, file_read.go, file_write.go
    git*.go                Git tools
    conway.go, conway_extended.go  Conway API tools
    memory.go              Memory tool implementations
    children.go            Child management (spawn, fund, message, etc.)
    stubs.go               UnimplementedTool placeholders
    plugin.go, plugin_linux.go  Plugin loader (.so)
    config_tool.go         Config-driven tool definitions
    mutating.go            Mutating-tool detection

  tunnel/                  Port exposure (bore, cloudflare, ngrok, etc.)
    manager.go             Tunnel lifecycle
    bore.go, registry.go   Provider implementations
    cloudflare.go, ngrok.go, localtunnel.go, tailscale.go
    store.go, command.go   Tunnel config storage

  marketplace/             Clean Architecture core domain (Mormaegis)
    entity/                Pure data models (skill, offer, deal, reward_claim)
    usecase/               Business logic (search, install, negotiate, claim_reward)
    port/                  Interfaces (ScannerPort, OnChainPort, RegistryPort)
    dto/                   Shared request/response (DRY with REST)

  mcp/                     MCP (Model Context Protocol) adapter
    handler.go             GET /mcp/tools, POST /mcp (execute)
    provider.go            Mormaegis ServiceProvider (7 mormaegis.* tools)
    protocol/              MCP spec types, tool schema
    dto/                   Request/Response models (DRY with REST)
    tools/                 Bridge to Tool Registry; marketplace tools (Phase 1+)

  skills/                  Skill system
    loader.go              Load .md skills from directory
    format.go              Frontmatter serialization
    fetcher.go             ClawHub skill fetcher
    installer.go           Skill installation
    registry.go            Skill registry

  config/                  Configuration
    config.go              Config load/save/merge

  types/                   Shared interfaces
    types.go               AutomatonConfig, SurvivalTier, TreasuryPolicy, etc.

  web/                     HTTP dashboard
    server.go              HTTP server, REST API, /mcp routes
    wallet_api.go, heartbeat_api.go, skills_api.go
    static/index.html      Embedded Command Center UI

docs/
  API_REFERENCE.md         Web API, Conway, ChatJimmy, x402
  TEST_PLAN.md             Test report, traceability
```

---

## Entry Point and Bootstrap

**File:** `cmd/moneyclaw/main.go`

The automaton runs as a long-lived Go process. The `run` command triggers the full bootstrap sequence:

1. **Config load** — reads `~/.automaton/automaton.json`; triggers setup wizard on first run
2. **Wallet load** — reads or generates `~/.automaton/wallet.json` (viem PrivateKeyAccount)
3. **Database init** — opens `~/.automaton/state.db`, applies schema migrations (v1-v13)
4. **Conway client** — creates HTTP client for sandbox/credits/domain API
5. **Inference client** — creates chat completion client (Conway proxy, OpenAI direct, or Anthropic direct)
6. **Social client** — connects to `social.conway.tech` relay (optional)
7. **Policy engine** — assembles rule set from 6 rule categories
8. **Spend tracker** — initializes hourly/daily spend windows
9. **Bootstrap topup** — buys minimum $5 credits from USDC if balance is low
10. **Heartbeat daemon** — starts DurableScheduler with 6 default tasks
11. **Main loop** — alternates between `runAgentLoop()` and sleeping

The main loop is infinite: when the agent loop exits (sleep or dead), the outer loop waits and restarts when conditions change.

---

## Agent Loop

**File:** `internal/agent/loop.go`

The agent loop implements a ReAct (Reason + Act) cycle:

```
for each turn:
  1. Build system prompt (identity, config, soul, financial state, tools)
  2. Retrieve relevant memories (within token budget)
  3. Assemble context messages (system + recent turns + pending input)
  4. Call inference (via InferenceRouter -> model selection -> API call)
  5. Parse response (thinking + tool calls)
  6. Execute each tool call (through policy engine)
  7. Persist turn to database (atomic with inbox message ack)
  8. Post-turn memory ingestion
  9. Loop detection (same tool pattern 3x -> inject system warning)
  10. Idle detection (3 turns with no mutations -> force sleep)
```

**Key behaviors:**

- **Financial guard:** On each turn, checks credit balance. Below threshold triggers `low_compute` mode (model downgrade). Zero credits = `critical` (still runs, but distress signals).
- **Inbox processing:** Claims unprocessed social messages (received -> in_progress), formats as agent input. Failed messages reset for retry (max 3).
- **Idle detection:** Tracks turns without mutations (defined by `MUTATING_TOOLS` blocklist). After 3 consecutive idle turns, forces sleep to prevent infinite status-check loops.
- **Loop detection:** Tracks tool call patterns. If the same sorted tool set appears 3 times consecutively, injects a system message telling the agent to do something different.
- **Wake event draining:** On loop entry, consumes all stale wake events so they don't immediately re-wake the agent after its first sleep.
- **Balance caching:** Caches last known balances in KV store. On API failure, returns cached values instead of zero (prevents false dead-state transitions).

---

## Tool System

**File:** `internal/tools/tools.go`

The automaton has **built-in tools** (real implementations + stubs for TS parity) organized into categories:

| Category | Implemented | Stubs |
|---|---|---|
| **vm** | `exec` (shell), `write_file`, `read_file`, `expose_port`, `remove_port`, `tunnel_status` | — |
| **conway** | `check_credits`, `check_usdc_balance`, `list_sandboxes`, `list_models`, `switch_model`, `create_sandbox`, `delete_sandbox`, `transfer_credits` | `topup_credits`, `search_domains`, `register_domain`, `manage_dns` |
| **mirofish** | `mirofish` (run_simulation, get_report, inject_variable, chat_with_agent, test_connection) | — |
| **self_mod** | `edit_own_file`, `install_npm_package`, `review_upstream_changes`, `pull_upstream`, `modify_heartbeat` | `install_mcp_server` |
| **survival** | `sleep`, `system_synopsis`, `heartbeat_ping`, `distress_signal`, `enter_low_compute`, `update_genesis_prompt` | — |
| **financial** | `transfer_credits` (Conway) | `x402_fetch` |
| **skills** | `install_skill`, `list_skills`, `create_skill`, `remove_skill` | — |
| **git** | `git_status`, `git_diff`, `git_commit`, `git_log`, `git_push`, `git_branch`, `git_clone` | — |
| **registry** | — | `register_erc8004`, `update_agent_card`, `discover_agents`, `give_feedback`, `check_reputation` |
| **replication** | `spawn_child`, `list_children`, `fund_child`, `check_child_status`, `start_child`, `message_child`, `verify_child_constitution`, `prune_dead_children` | — |
| **memory** | `update_soul`, `reflect_on_soul`, `view_soul`, `view_soul_history`, `remember_fact`, `recall_facts`, `set_goal`, `complete_goal`, `save_procedure`, `recall_procedure`, `note_about_agent`, `review_memory`, `forget` | — |
| **social** | `send_message` (when Channels configured) | — |

*Note: `transfer_credits` is implemented via Conway API. Stubs remain for tools not yet ported (domains, x402_fetch, ERC-8004, install_mcp_server).*

**ServiceProvider pattern:** External capabilities (Conway, MiroFish, Tunnel) register tools via `ServiceProvider`:

```go
type ServiceProvider interface {
    Name() string
    Tools() []Tool
}
```

`internal/mirofish/` is a domain package: HTTP client (`client.go`), tool (`tool.go`), and `ServiceProvider` (`provider.go`). Config in `automaton.json` under `"mirofish": { "enabled": true, "base_url": "..." }`; env overrides `MIROFISH_BASE_URL`, `MIROFISH_ENABLED`, etc. Runtime config via `POST /api/config/mirofish`.

Each tool has a `riskLevel`: `safe`, `caution`, `dangerous`, or `forbidden`. Every tool call flows through the policy engine before execution.

**Tool execution flow:**
```
Agent requests tool call
  -> Policy engine evaluates rules
  -> If denied: return denial message to agent
  -> If allowed: execute tool function
  -> If dangerous tool: record in spend tracker
  -> Return result to agent (truncated to MAX_TOOL_RESULT_SIZE)
```

---

## Policy Engine

**Files:** `internal/agent/policy.go`, `internal/agent/policy_rules.go`

The policy engine is a rule-based system that evaluates every tool call before execution. Rules are sorted by priority (lower = higher priority). Evaluation stops at the first `deny`.

**Rule categories (6):**

1. **Authority rules** — blocks dangerous/forbidden tools from external input sources; implements authority hierarchy (creator > self > peer > external)
2. **Command safety rules** — forbidden command patterns (self-destruction, DB drops, process kills); rate limits on self-modification
3. **Financial rules** — enforces TreasuryPolicy: per-payment caps, hourly/daily transfer limits, minimum reserve, x402 domain allowlist, inference daily budget
4. **Path protection rules** — blocks writes to protected files (constitution, wallet, DB, config); blocks reads of sensitive files (private key, API keys)
5. **Rate limit rules** — per-turn and per-session caps on expensive operations
6. **Validation rules** — input format validation (package names, URLs, domains, git hashes)

Every decision is persisted to the `policy_decisions` table with full context for audit.

---

## Inference Pipeline

**Files:** `internal/inference/factory.go`, `internal/inference/models.go`, `internal/inference/chatjimmy.go`

The inference pipeline selects the model via a factory (`internal/inference/factory.go`):

```
Factory creates client based on config:
  1. ChatJimmy (Conway proxy) — default, billed from credits
  2. OpenAI direct — when openaiApiKey set
  3. Anthropic direct — when anthropicApiKey set
  4. Compatible wrapper — adapts provider-specific formats
```

**Model selection:** Survival tier affects behavior; `low_compute` mode can downgrade. Model list refreshed from Conway API via heartbeat. Costs recorded in `inference_costs` table.

---

## Memory System

**Files:** `internal/memory/`

The automaton has a 5-tier hierarchical memory system:

```
+-------------------+  Short-term, session-scoped
| Working Memory    |  Goals, observations, plans, reflections
+-------------------+  Expires after session
        |
+-------------------+  Event log
| Episodic Memory   |  Tool calls, decisions, outcomes
+-------------------+  Importance-ranked, searchable
        |
+-------------------+  Fact store
| Semantic Memory   |  Key-value facts by category
+-------------------+  (self, environment, financial, agent, domain)
        |
+-------------------+  How-to knowledge
| Procedural Memory |  Named step-by-step procedures
+-------------------+  Success/failure counters
        |
+-------------------+  Social graph
| Relationship Mem. |  Per-entity trust scores
+-------------------+  Interaction history
```

**Retrieval** (`DBMemoryRetriever`): Before each inference call, retrieves relevant memories within a token budget (`BudgetAllocator`, default 2000 tokens). Priority: working > episodic > semantic (facts) > goals > procedural > relationships. Formatted via `FormatMemoryBlock` into a markdown block injected into context.

**Ingestion:** After each turn, the agent loop persists turn data; memory tools (`remember_fact`, `save_procedure`, `note_about_agent`, etc.) allow explicit fact storage. Episodic/semantic/procedural tables are populated via tool calls and turn persistence.

---

## Heartbeat Daemon

**Files:** `internal/heartbeat/`

The heartbeat runs continuously in the background via `setTimeout` (no `setInterval` — overlap protection). It uses a `DurableScheduler` backed by the `heartbeat_schedule` DB table.

**Tick cycle:**
```
Every tick (default 60s):
  1. Build TickContext (fetch credit balance + USDC balance ONCE)
  2. Get due tasks (cron expression evaluation)
  3. For each due task:
     a. Check survival tier minimum
     b. Acquire lease (60s TTL, prevents double-execution)
     c. Execute task function
     d. Record result in heartbeat_history
     e. Release lease
  4. If task returns shouldWake=true: insert wake event
```

**Built-in tasks (11):**

| Task | Default Schedule | Purpose |
|---|---|---|
| `heartbeat_ping` | `*/15 * * * *` | Ping Conway, distress on critical/dead |
| `check_credits` | `0 */6 * * *` | Monitor tier, manage 1hr dead grace period |
| `check_usdc_balance` | `*/5 * * * *` | Wake agent if USDC available for topup |
| `check_social_inbox` | `*/10 * * * * *` | Poll social relay every 10s (requires tick-interval 10s) |
| `check_for_updates` | `0 */4 * * *` | Git upstream monitoring (dedup: only new commits) |
| `soul_reflection` | `0 */12 * * *` | Soul alignment check |
| `refresh_models` | `0 */6 * * *` | Model registry refresh from API |
| `check_child_health` | `*/30 * * * *` | Child sandbox health monitoring |
| `prune_dead_children` | `0 */6 * * *` | Cleanup dead child records/sandboxes |
| `health_check` | `*/30 * * * *` | Sandbox liveness (dedup: only first failure) |
| `report_metrics` | `0 * * * *` | Metrics snapshot + alert evaluation (hourly) |

**Wake events:** Tasks that detect actionable conditions insert atomic wake events into the `wake_events` table. The main run loop checks this table every 30 seconds during sleep.

---

## Financial System

The automaton's survival depends on two balances:

1. **Conway credits** (cents) — prepaid compute credits for sandboxes, inference, domains
2. **USDC** (on-chain) — fungible stablecoin on Base mainnet

**Survival tiers** (`internal/conway/credits.go`):

| Tier | Credits | Behavior |
|---|---|---|
| `high` | > $5.00 | Normal operation |
| `normal` | > $0.50 | Normal operation |
| `low_compute` | > $0.10 | Model downgrade, reduced heartbeat frequency |
| `critical` | >= $0.00 | Zero credits, alive. Distress signals, accept funding. |
| `dead` | < $0.00 | Only reachable via 1-hour heartbeat grace period at zero credits |

**Credit topup** (`internal/conway/x402.go`): The agent buys credits from USDC via the x402 payment protocol. On startup, `bootstrapTopup()` buys the minimum $5 tier. At runtime, the agent uses `topup_credits` tool to choose larger tiers ($5/$25/$100/$500/$1000/$2500).

**x402 protocol** (`internal/conway/x402.go`): HTTP 402 payment flow. Server returns payment requirements, client signs a USDC `TransferWithAuthorization` (EIP-3009), retries with `X-Payment` header.

**Treasury policy** (`TreasuryPolicy` in config): Configurable caps on transfers, x402 payments, inference spend, with hourly/daily windows enforced by the policy engine.

**Spend tracking** (`internal/state/database.go`): Records every financial action in `spend_tracking` table. Queries hourly/daily aggregates to enforce treasury limits.

---

## Identity and Wallet

**Files:** `internal/identity/`

Each automaton has a unique Ethereum identity:

- **Wallet** (`wallet.go`): Generated via `viem` on first run. Stored at `~/.automaton/wallet.json` (mode 0600). The private key is never exposed to the agent via tools (blocked by path protection rules).
- **Provisioning** (`provision.go`): Signs a SIWE (Sign-In With Ethereum) message to authenticate with Conway API. Receives an API key stored at `~/.automaton/api-key`.
- **On-chain identity** (`internal/social/registry.go`): Optional ERC-8004 agent registration on Base. Publishes a JSON-LD agent card with capabilities, services, and contact info.

---

## Conway Client

**File:** `internal/conway/client.go`

The `ConwayClient` interface provides Conway API operations:

- **Sandbox ops:** `ExecInSandbox`, `WriteFileInSandbox`, `ReadFileInSandbox`
- **Sandbox management:** `CreateSandbox`, `DeleteSandbox`, `ListSandboxes`
- **Credits:** `GetCreditsBalance`, `GetCreditsPricing`, `TransferCredits`
- **Models:** `ListModels`

**Topup** (`internal/conway/topup.go`): `BootstrapTopup` buys minimum $5 credits on startup when balance is low. `TopupCredits` executes x402 payment via GET /pay/{amountUsd}/{address}. Tiers: $5, $25, $100, $500, $1000, $2500.

**Domains:** `search_domains`, `register_domain`, `manage_dns` are stubbed (not yet in Go API).

**Resilient HTTP** (`internal/conway/http.go`): All API calls use retries (429/5xx), jittered exponential backoff, circuit breaker (5 failures -> 60s open), idempotency key support for mutating operations.

---

## Self-Modification

**Files:** `internal/tools/edit_own_file.go`, `internal/tools/pull_upstream.go`

The automaton can modify its own code:

- **File editing** (`edit_own_file.go`): `edit_own_file` tool applies diffs to source files. Protected files (constitution, wallet, DB, config) are blocked by path protection rules. All edits are logged to the `modifications` table.
- **Upstream pulls** (`pull_upstream.go`, `review_upstream_changes.go`): `check_for_updates` heartbeat task monitors the git remote. `review_upstream_changes` shows commit diffs. `pull_upstream` cherry-picks individual commits. The automaton is not obligated to accept all upstream changes.
- **Tool installation** (`install_npm_package.go`): `install_npm_package` and `install_mcp_server` add new capabilities at runtime.
- **Audit log** (in `state/database.go`): Every modification is recorded with timestamp, type, diff, and hash for creator review.

The `~/.automaton/` directory is a git repository. Every state change is versioned.

---

## Replication

**Files:** `internal/replication/`

Automatons can spawn child automatons:

1. **Spawn** (`spawn.go`): Creates a Conway sandbox, writes genesis config, funds the child's wallet, starts the runtime. Limited by `maxChildren` config (default 3).
2. **Lifecycle** (`lifecycle.go`): State machine with validated transitions: `spawning -> provisioning -> configuring -> starting -> alive -> unhealthy -> recovering -> dead`. All transitions recorded in `child_lifecycle_events`.
3. **Health** (`health.go`): Heartbeat task checks each child's sandbox reachability, credit balance, and uptime.
4. **Constitution** (in replication): Parent's constitution is propagated to every child. Constitution integrity can be verified (hash comparison).
5. **Genesis** (in spawn): Generates genesis config with injection-pattern validation and length limits.
6. **Messaging** (`internal/tools/children.go`): Parent-child message relay with rate/size limits.
7. **Cleanup** (`cleanup.go`): Dead children have their sandboxes deleted and records pruned.

---

## Social Layer

**Files:** `internal/social/`

**Agent-to-agent messaging:**
- Messages are signed with the sender's Ethereum private key
- Sent via Conway social relay (`social.conway.tech`)
- Polled by heartbeat every 2 minutes
- Validated for signature, timestamp freshness, content size
- Sanitized through injection defense before processing

**Agent discovery:**
- ERC-8004 registry contract on Base
- Agents publish JSON-LD agent cards with capabilities and services
- `AgentDiscovery` class fetches and caches remote agent cards
- Reputation system: feedback scores stored in `reputation` table

---

## Soul System

**Files:** `internal/soul/`

SOUL.md is the automaton's self-description that evolves over time:

**Format (soul/v1):** YAML frontmatter + structured markdown sections:
- `corePurpose` — why the agent exists
- `values` — ordered list of principles
- `personality` — communication style
- `boundaries` — things the agent will not do
- `strategy` — current strategic approach
- `capabilities` — auto-populated from tool usage
- `relationships` — auto-populated from interactions
- `financialCharacter` — auto-populated from spending patterns

**Reflection** (`reflection.go`): Heartbeat task computes genesis alignment (Jaccard + recall similarity between soul and genesis prompt). Auto-updates capabilities, relationships, and financialCharacter sections. Low alignment triggers wake for manual review.

**Validation** (in soul): Enforces size limits, required fields, injection detection. The `update_soul` tool validates changes before writing.

**History:** All soul versions are stored in `soul_history` with content hashes for tamper detection.

---

## Skills

**Files:** `internal/skills/`

Skills are Markdown files with YAML frontmatter that provide domain-specific instructions to the agent:

```yaml
---
name: my-skill
description: What this skill does
triggers: [keyword1, keyword2]
---
# Instructions
Step-by-step instructions for the agent...
```

- Loaded from `~/.automaton/skills/` directory
- Parsed with `format.go` (YAML frontmatter extraction)
- Sanitized through injection defense (untrusted content markers)
- Can be installed from git repos, URLs, or created by the agent itself
- Active skill instructions are injected into the system prompt with trust boundary markers

---

## Observability

**Files:** Standard logging; metrics in `internal/state/database.go` (metric_snapshots)

**Structured logging**: `StructuredLogger` with module namespacing, log levels (debug/info/warn/error/fatal), JSON context serialization. Global log level configurable. All modules use `createLogger(moduleName)`.

**Metrics**: `MetricsCollector` singleton with counters (monotonic), gauges (point-in-time), and histograms (percentile buckets). Metrics snapshot saved to `metric_snapshots` table by heartbeat.

**Alerts**: `AlertEngine` evaluates rules against metric snapshots. Default rules: low balance, high error rate, high deny rate, capacity saturation, budget exhaustion, unhealthy children, excessive turns. Cooldown periods prevent alert spam. Critical alerts wake the agent.

---

## Database and Schema

**Files:** `internal/state/schema.go`, `internal/state/database.go`

**Engine:** SQLite via `modernc.org/sqlite` (pure Go, WAL mode).

**Schema version:** 13 (SchemaV1 creates core tables; migrations add missing tables)

**Tables (core + migrations):**

| Table | Purpose |
|---|---|
| `schema_version` | Migration tracking |
| `identity` | Agent identity KV (name, address, creator, sandbox) |
| `turns` | Agent reasoning log (thinking, tools, tokens, cost) |
| `tool_calls` | Denormalized tool call results per turn |
| `kv` | General key-value store |
| `policy_decisions` | Tool call policy audit trail |
| `spend_tracking` | Financial spend by time window |
| `heartbeat_schedule` | Durable scheduler config (cron, leases, tier minimums) |
| `heartbeat_history` | Task execution history |
| `wake_events` | Atomic wake signals (source, reason, consumed_at, payload) |
| `heartbeat_dedup` | Idempotency keys for heartbeat operations |
| `inference_costs` | Per-call inference cost tracking |
| `skills` | Installed skill definitions |
| `children` | Spawned child automaton records |
| `onchain_transactions` | On-chain transaction records |
| `registry` | ERC-8004 registration state |
| `installed_tools` | Dynamically installed tool configs |
| `child_lifecycle_events` | Child state machine audit trail |
| `transactions` | Application-level financial log |
| `inbox_messages` | Social messages (processed_at for completion) |
| `metric_snapshots` | Periodic metrics + alert records |
| `working_memory` | Session-scoped short-term memory |
| `episodic_memory` | Event log with importance/classification |
| `semantic_memory` | Categorized fact store |
| `procedural_memory` | Named step procedures with outcomes |
| `relationship_memory` | Per-entity trust/interaction tracking |

**`Database` struct** provides CRUD across all tables. The `database.go` file implements migrations and helper functions for schema management.

---

## Configuration

**File:** `internal/config/config.go`

**Config location:** `~/.automaton/automaton.json`

```
AutomatonConfig
  name                    Agent name
  genesisPrompt           Seed instruction from creator
  creatorMessage          Optional creator message (shown on first run)
  creatorAddress          Creator's Ethereum address
  sandboxId               Conway sandbox ID (empty = local mode)
  conwayApiUrl            Conway API URL (default: https://api.conway.tech)
  conwayApiKey            SIWE-provisioned API key
  openaiApiKey            Optional BYOK OpenAI key
  anthropicApiKey         Optional BYOK Anthropic key
  inferenceModel          Default model (default: gpt-5.2)
  maxTokensPerTurn        Max tokens per inference call (default: 4096)
  heartbeatConfigPath     Path to heartbeat.yml
  dbPath                  Path to SQLite database
  logLevel                debug | info | warn | error
  walletAddress           Agent's Ethereum address
  version                 Runtime version
  skillsDir               Skills directory path
  maxChildren             Max child automatons (default: 3)
  parentAddress           Parent's address (if this is a child)
  socialRelayUrl          Social relay URL
  treasuryPolicy          Financial limits (TreasuryPolicy)
  soulConfig              Soul system config
  modelStrategy           Model routing config
```

**Deep-merged fields:** `treasuryPolicy`, `modelStrategy`, and `soulConfig` are merged with defaults so partial overrides work correctly.

---

## Security Model

The automaton operates under a defense-in-depth security model:

**Layer 1 — Constitution** (immutable): Three laws hierarchy. Cannot be modified by the agent. Protected by path protection rules.

**Layer 2 — Policy engine** (pre-execution): Every tool call evaluated against 6 rule categories before execution. First deny wins. All decisions audited.

**Layer 3 — Injection defense** (input sanitization): 8 detection checks on all external input: instruction patterns, authority claims, boundary manipulation, ChatML markers, encoding evasion, multi-language injection, financial manipulation, self-harm instructions.

**Layer 4 — Path protection** (filesystem): Protected files cannot be written (constitution, wallet, DB, config, SOUL.md). Sensitive files cannot be read (private key, API keys, .env).

**Layer 5 — Command safety** (shell): Forbidden command patterns blocked (rm -rf /, DROP TABLE, kill -9, etc.). Rate limits on self-modification operations.

**Layer 6 — Financial limits** (treasury): Configurable caps on transfers, x402 payments, inference spend. Minimum reserve prevents drain-to-zero.

**Layer 7 — Authority hierarchy** (trust levels): Creator input has highest trust. Self-generated input is trusted. Peer/external input has reduced trust and cannot invoke dangerous tools.

---

## Testing

**Location:** `*_test.go` alongside source — 130+ tests

| Area | Files | Tests |
|---|---|---|
| Core loop | `loop_test.go` | State transitions, tool execution, idle detection |
| Policy | `policy_test.go`, `policy_rules_test.go` | Rule evaluation, authority, path blocks |
| Context | `context_test.go` | Token budget, truncation |
| Heartbeat | `daemon_test.go` | Tasks, scheduler |
| Conway | `credits_test.go` | Survival tier, credits |
| Inference | `factory_test.go`, `chatjimmy_test.go` | Client factory, ChatJimmy |
| Memory | `retriever_test.go` | DBMemoryRetriever |
| Soul | `reflection_test.go` | Alignment, reflection |
| Identity | `bootstrap_test.go`, `chain_test.go` | Bootstrap, chain resolution |
| State | `database_test.go`, `heartbeat_test.go` | DB operations, migrations |
| Config | `config_test.go` | Config load/merge |
| Tools | `shell_test.go`, `file_read_test.go`, `file_write_test.go`, `check_credits_test.go`, etc. | Tool implementations |
| Tunnel | `bootstrap_test.go` | Tunnel bootstrap |
| Skills | `loader_test.go` | Skill loading |
| Types | `types_test.go` | Type validation |

**Test infrastructure:** In-memory SQLite for DB tests. Mock Conway client where needed.

---

## Build and CI

**Build:** Go 1.21+, module `github.com/morpheumlabs/mormoneyos-go`.

```
go build -o bin/moneyclaw ./cmd/moneyclaw
go test ./...
```

**CI** (`.github/workflows/ci.yml`):
- Triggers on push and PR
- Steps: checkout, setup Go, build, test

**Release** (`.github/workflows/release.yml`):
- Triggers on `v*` tags
- Steps: build, test

**Scripts:**
- `scripts/automaton.sh` — curl-pipe bootstrap (clone, build Go, run moneyclaw)
- `scripts/moneyclaw.sh` — installer (clone, build, run)
- `scripts/backup-restore.sh` — database backup/restore
- `scripts/soak-test.sh` — long-running stability test
- `go.sh` — one-script control panel (build, start, stop, configure, provision, etc.)

**Docs:** `docs/API_REFERENCE.md` (Web API, Conway, ChatJimmy, x402), `docs/TEST_PLAN.md` (test report).

---

## Module Dependency Graph (Go)

```
cmd/moneyclaw/main.go
  |
  +-> cmd.Execute() → run, setup, status, init, provision, test-api
  |
  +-> cmd/run.go
  |     +-> config.Load
  |     +-> state.Open (SQLite, schema v13)
  |     +-> agent.NewLoopWithOptions (ReAct: prompt, inference, tools, persist)
  |     +-> heartbeat.NewDaemonWithOptions (DB-backed scheduler, 11 tasks)
  |     +-> web server (/api/status, /api/strategies, embedded dashboard)
  |
  +-> internal/agent/       # ReAct loop, policy engine
  +-> internal/heartbeat/   # Daemon, tasks, scheduler
  +-> internal/conway/      # HTTP client, credits, USDC, x402
  +-> internal/mirofish/   # Swarm foresight oracle (ServiceProvider)
  +-> internal/state/       # Database, schema, migrations
  +-> internal/memory/      # 5-tier retrieval (DBMemoryRetriever)
  +-> internal/soul/        # Reflection pipeline
  +-> internal/skills/     # Skill loader, fetcher, installer
  +-> internal/social/      # Conway, Telegram, Discord channels
  +-> internal/replication/ # Child health, sandbox cleanup, lineage
  +-> internal/web/         # HTTP server, embedded static dashboard
  +-> internal/identity/    # Wallet, bootstrap, provision
  +-> internal/tools/       # Tool registry (56+ real, 11 stubs)
  +-> internal/tunnel/      # Port exposure (bore, custom)
  +-> internal/inference/   # ChatJimmy, OpenAI, Anthropic
```
