# mormoneyOS API Reference

**Date:** 2026-03-13  
**Purpose:** Complete documentation of all supported API paths across the mormoneyOS system.

---

## 1. Overview

mormoneyOS integrates with several API surfaces:

| Surface | Hosted By | Purpose |
|---------|-----------|---------|
| **Web Dashboard API** | mormoneyOS (`moneyclaw run`) | Agent status, control, dashboard |
| **Conway API** | Conway (external) | Credits, sandboxes, inference |
| **ChatJimmy API** | ChatJimmy (external) | Inference when `provider: chatjimmy` |
| **Conway Auth** | Conway (external) | SIWE provisioning, API keys |
| **Conway x402** | Conway (external) | USDC credit topup |

---

## 2. Web Dashboard API (mormoneyOS Hosted)

**Base URL:** `http://<web-addr>` (default `:8080`)  
**Config:** `--web-addr`, `--no-web` to disable  
**Source:** `internal/web/server.go`

### 2.1 Static & Dashboard

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Dashboard HTML (embedded `static/index.html`) |
| GET | `/static/*` | Static assets (embedded from `static/`) |

### 2.2 REST API Endpoints

| Method | Path | Status | Description | Response Shape |
|--------|------|--------|-------------|----------------|
| GET | `/api/status` | ✅ Implemented | Agent runtime status, credits, identity | `{ is_running, state, tick_count, wallet_value, today_pnl, dry_run, address, chain, name, version, running, paused, agent_state, tick }` |
| GET | `/api/strategies` | ✅ Implemented | Skills + children from DB (or hardcoded fallback) | `[{ name, description, risk_level, enabled }, ...]` |
| GET | `/api/history` | ⚠️ Placeholder | Memory/history — returns `[]` | `[]` |
| GET | `/api/cost` | ✅ Implemented | Inference cost summary (when `inference_costs` exists) | `{ today_cost, today_calls, total_cost, over_budget, by_layer, calls_by_layer }` |
| GET | `/api/risk` | ✅ Implemented | Risk state from RuntimeState | `{ paused, daily_loss, risk_level }` |
| POST | `/api/pause` | ✅ Implemented | Pause agent (set sleeping, sleep_until far future) | `{ status: "paused" }` |
| POST | `/api/resume` | ✅ Implemented | Resume agent (delete sleep_until, insert wake event) | `{ status: "resumed" }` |
| POST | `/api/chat` | ⚠️ Keyword-only | Simple chat (status/help parsing); frontend shows "not yet implemented" | `{ response: string }` |

### 2.3 Query Parameters

- **GET /api/status**: `?chain=<CAIP-2>` — Override chain for address resolution (e.g. `eip155:8453`).

### 2.4 POST /api/chat Request Body

```json
{ "message": "status" }
```

Supported messages: `status`, `help`, `帮助` — others return a generic "I don't understand" response.

---

## 3. Conway API (External Client)

**Base URL:** `conwayApiUrl` from config (e.g. `https://api.conway.tech`)  
**Auth:** `Authorization: <conwayApiKey>`  
**Source:** `internal/conway/http.go`

### 3.1 Credits

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/credits/balance` | Get credits balance (cents) |
| GET | `/v1/credits/pricing` | Get model pricing (404-tolerant) |
| POST | `/v1/credits/transfer` | Transfer credits to address (or `/v1/credits/transfers` on 404) |

### 3.2 Sandboxes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/sandboxes` | List sandboxes |
| POST | `/v1/sandboxes` | Create sandbox |
| POST | `/v1/sandboxes/{id}/exec` | Run command in sandbox |
| POST | `/v1/sandboxes/{id}/files/upload/json` | Write file in sandbox |
| GET | `/v1/sandboxes/{id}/files/read?path=<path>` | Read file from sandbox |

**Note:** `DeleteSandbox` is a no-op; Conway no longer supports sandbox deletion.

### 3.3 Models

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/models` | List available models |

---

## 4. Conway Auth (External – SIWE Provisioning)

**Base URL:** `https://api.conway.tech` (or `conwayApiUrl`)  
**Source:** `internal/identity/provision.go`

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/auth/nonce` | Get SIWE nonce |
| POST | `/v1/auth/verify` | Verify SIWE signature → `access_token` |
| POST | `/v1/auth/api-keys` | Create API key (Bearer token required) |
| POST | `/v1/automaton/register-parent` | Register creator address (404-tolerant) |

---

## 5. Conway x402 Topup (External)

**Base URL:** `conwayApiUrl`  
**Source:** `internal/conway/topup.go`, `internal/conway/x402.go`

| Method | Path | Description |
|--------|------|-------------|
| GET | `/pay/{amountUsd}/{address}` | x402 payment; returns 402 with `X-Payment-Required`; client signs USDC TransferWithAuthorization and retries with `X-Payment` header |

**Topup tiers (USD):** 5, 25, 100, 500, 1000, 2500

---

## 6. ChatJimmy API (External – Inference)

**Base URL:** `chatJimmyApiUrl` or `CHATJIMMY_BASE_URL` (default `https://chatjimmy.ai`)  
**Source:** `internal/inference/chatjimmy.go`

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/health` | Health check (status + backend) |
| GET | `/api/models` | List available models |
| POST | `/api/chat` | Chat completion |

**Note:** ChatJimmy endpoints are used when `provider: chatjimmy`; they are not part of the mormoneyOS web dashboard.

---

## 7. Summary Table

| Category | Paths |
|----------|-------|
| **Web Dashboard** | `/`, `/static/*`, `GET /api/status`, `GET /api/strategies`, `GET /api/history`, `GET /api/cost`, `GET /api/risk`, `POST /api/pause`, `POST /api/resume`, `POST /api/chat` |
| **Conway** | `GET/POST /v1/credits/*`, `GET/POST /v1/sandboxes`, `POST /v1/sandboxes/{id}/exec`, `POST /v1/sandboxes/{id}/files/upload/json`, `GET /v1/sandboxes/{id}/files/read`, `GET /v1/models` |
| **Conway Auth** | `POST /v1/auth/nonce`, `POST /v1/auth/verify`, `POST /v1/auth/api-keys`, `POST /v1/automaton/register-parent` |
| **Conway x402** | `GET /pay/{amountUsd}/{address}` |
| **ChatJimmy** | `GET /api/health`, `GET /api/models`, `POST /api/chat` |

---

## 8. internal/web Implementation Status & Extensions

### 8.1 Current State

| Endpoint | Status | Notes |
|---------|--------|-------|
| `GET /api/status` | ✅ Full | DB + Conway credits, identity, turn count |
| `GET /api/strategies` | ✅ Full | Skills + children from DB; hardcoded fallback when empty |
| `GET /api/history` | ⚠️ Placeholder | Returns `[]`; no data wired |
| `GET /api/cost` | ✅ Full | `state.GetInferenceCostSummary()` when DB has `inference_costs` |
| `GET /api/risk` | ✅ Full | RuntimeState.Paused + static risk_level |
| `POST /api/pause` | ✅ Full | DB: SetAgentState + SetKV sleep_until |
| `POST /api/resume` | ✅ Full | DB: DeleteKV + InsertWakeEvent |
| `POST /api/chat` | ⚠️ Keyword-only | status/help/帮助; frontend does not call it (shows "not yet implemented") |

**Auth:** None. Dashboard is unauthenticated. Anyone reaching the URL has full access.

### 8.2 Implementable in internal/web

Endpoints that can be added using existing data sources. Requires extending `Server` with optional dependencies (type-assert `WebDB` to `*state.Database`, pass `Tools`, `TunnelManager`, `Config`, etc.).

| Endpoint | Data Source | Effort | Description |
|----------|-------------|--------|-------------|
| `GET /api/history` | `state.GetRecentTurns(limit)` | Low | Turn history: id, timestamp, state, input, thinking, tool_calls, cost_cents. Extend WebDB or type-assert to `*state.Database`. |
| `GET /api/soul` | `DB.GetKV("soul_content")` | Low | Read soul document from KV. Same pattern as `view_soul` tool. |
| `GET /api/memory` | `state.GetSemanticMemory`, `GetProceduralMemory`, KV goals/facts | Medium | Facts, goals, procedures. Requires schema v13 (5-tier) + KV keys like `goal:`, `procedure:`. |
| `GET /api/health` | — | Trivial | Liveness: return `200 OK` or `{"ok": true}`. |
| `GET /api/tools` | `tools.Registry.List()` | Low | List registered tool names. Pass `Tools` (Registry) to Server. |
| `GET /api/config` | `*types.AutomatonConfig` (sanitized) | Low | Read-only config summary: name, chain, version. **Never expose** API keys, wallet paths. |
| `GET /api/heartbeat` | `state.GetHeartbeatSchedule`, heartbeat history | Medium | Schedule + recent run history. |
| `GET /api/tunnels` | `tunnel.TunnelManager.Status()` | Low | Active tunnels (port, provider, public_url). Pass TunnelManager to Server. |
| `POST /api/chat` (enhance) | Inference client or agent loop | High | Wire to real LLM or forward to agent. Frontend must call `POST /api/chat` with body `{ "message": "..." }`. |

### 8.3 Frontend Gaps

- **Chat:** `static/index.html` `btnSend` handler does **not** call `POST /api/chat`. It appends "(Chat API not yet implemented)". Wire `fetch('/api/chat', { method: 'POST', body: JSON.stringify({ message }) })` to use the existing keyword handler.
- **History:** No UI for `/api/history`; add a panel when endpoint returns real data.

---

## 9. References

- [ts-go-alignment.md](design/ts-go-alignment.md) — Web API alignment with TypeScript reference
- [child-runtime-protocol.md](design/child-runtime-protocol.md) — Conway sandbox operations
- [wallet-identity-architecture.md](design/wallet-identity-architecture.md) — SIWE provisioning flow
