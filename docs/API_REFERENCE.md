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

| Method | Path | Description | Response Shape |
|--------|------|-------------|----------------|
| GET | `/api/status` | Agent runtime status, credits, identity | `{ is_running, state, tick_count, wallet_value, today_pnl, dry_run, address, chain, name, version, running, paused, agent_state, tick }` |
| GET | `/api/strategies` | Skills + children from DB (or hardcoded fallback) | `[{ name, description, risk_level, enabled }, ...]` |
| GET | `/api/history` | Memory/history (placeholder) | `[]` |
| GET | `/api/cost` | Inference cost summary | `{ today_cost, today_calls, total_cost, over_budget, by_layer, calls_by_layer }` |
| GET | `/api/risk` | Risk state | `{ paused, daily_loss, risk_level }` |
| POST | `/api/pause` | Pause agent (set sleeping, sleep_until far future) | `{ status: "paused" }` |
| POST | `/api/resume` | Resume agent (delete sleep_until, insert wake event) | `{ status: "resumed" }` |
| POST | `/api/chat` | Simple chat (status/help parsing) | `{ response: string }` |

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

## 8. References

- [ts-go-alignment.md](design/ts-go-alignment.md) — Web API alignment with TypeScript reference
- [child-runtime-protocol.md](design/child-runtime-protocol.md) — Conway sandbox operations
- [wallet-identity-architecture.md](design/wallet-identity-architecture.md) — SIWE provisioning flow
