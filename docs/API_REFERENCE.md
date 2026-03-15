# mormoneyOS API Reference

**Date:** 2026-03-14  
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
| POST | `/api/chat` | ⚠️ Keyword-only | Simple chat (status/help parsing) | `{ response: string }` |
| GET | `/api/config` | ✅ Implemented | Read automaton.json config (raw content) | `{ content: string }` |
| PUT | `/api/config` | ✅ Implemented | Write automaton.json config (full JSON body) | `204 No Content` |
| GET | `/api/soul/config` | ✅ Implemented | Read soul config (personality, system prompt, tone, constraints) | `{ systemPrompt, personality, tone, behavioralConstraints }` |
| PUT | `/api/soul/config` | ✅ Implemented | Update soul config (partial merge) | `204 No Content` |
| GET | `/api/tools` | ✅ Implemented | List registered tools with enabled state (requires ToolsLister) | `{ tools: [{ name, description, enabled }] }` |
| PATCH | `/api/tools/{name}` | ✅ Implemented | Toggle tool enabled/disabled | `{ name, enabled }` |
| GET | `/api/social` | ✅ Implemented | List social channels with status and config schema | `{ channels: [{ name, displayName, enabled, ready, configFields, config }] }` |
| PATCH | `/api/social/{name}` | ✅ Implemented | Toggle channel enabled/disabled | `{ name, enabled }` |
| PUT | `/api/social/{name}/config` | ✅ Implemented | Update channel config (validates before save) | `{ ok, validated?, enabled?, error? }` |
| POST | `/api/auth/verify` | ✅ Implemented | Verify message signature (Ethereum, Solana, Bitcoin, Morpheum) | `{ valid: bool, address?: string, error?: string }` |
| GET | `/api/reports` | ✅ Implemented | Metric reports (last_metrics_report + metric_snapshots) | `{ last_report?, snapshots: [{ id, snapshot_at, metrics, alerts }] }` |
| GET | `/api/tunnels` | ✅ Implemented | Active tunnels (port, provider, public_url) | `{ tunnels: [{ port, provider, public_url }] }` |
| GET | `/api/tunnels/providers` | ✅ Implemented | Provider schemas + config (secrets masked) | `{ providers, schemas, config }` |
| PUT | `/api/tunnels/providers/{name}` | ✅ Implemented | Update provider config (token, authToken, authKey, etc.) in automaton.json | `{ ok, provider }` |
| POST | `/api/tunnels/providers/{name}/restart` | ✅ Implemented | Reload provider (requires API key in automaton.json for cloudflare/ngrok/tailscale) | `{ ok, provider, restarted }` |
| GET | `/api/models` | ✅ Implemented | List configured LLM models (API keys masked) + providers | `{ models: [{ id, provider, modelId, apiKeyMasked, contextLimit, costCapCents, priority, enabled }], providers: [{ key, displayName, local }] }` |
| POST | `/api/models` | ✅ Implemented | Add a model | `201` + model object |
| PATCH | `/api/models/{id}` | ✅ Implemented | Update model (apiKey, modelId, contextLimit, costCapCents, enabled) | Model object |
| DELETE | `/api/models/{id}` | ✅ Implemented | Remove a model | `204 No Content` |
| PUT | `/api/models/order` | ✅ Implemented | Reorder models by priority | `204 No Content` |
| GET | `/api/skills` | ✅ Implemented | List installed skills (filter, trusted) | `{ skills: [{ name, description, source, path, enabled, trusted }] }` |
| GET | `/api/skills/discovery` | ✅ Implemented | Discover skills from ClawHub (search or list) | `{ results?: [...] }` or `{ items, nextCursor }` |
| GET | `/api/skills/recommended` | ✅ Implemented | Recommended skills from ClawHub | `{ recommended: [{ slug, displayName, summary, version, installed }] }` |
| GET | `/api/skills/{name}` | ✅ Implemented | Get single skill | `{ name, description, source, path, enabled, trusted }` |
| POST | `/api/skills` | ✅ Implemented | Install skill (path or ClawHub) | `201` + skill object |
| PATCH | `/api/skills/{name}` | ✅ Implemented | Update skill (enabled, description) | Skill object |
| DELETE | `/api/skills/{name}` | ✅ Implemented | Remove skill | `204 No Content` |
| PATCH | `/api/skills/{name}/activate` | ✅ Implemented | Enable skill | `{ name, enabled: true }` |
| PATCH | `/api/skills/{name}/deactivate` | ✅ Implemented | Disable skill | `{ name, enabled: false }` |
| GET | `/api/heartbeat` | ✅ Implemented | List all heartbeat_schedule rows from DB | `{ schedules: [{ name, schedule, task, enabled, tierMinimum, lastRun, nextRun, leaseUntil, leaseOwner }] }` |
| PATCH | `/api/heartbeat/{name}` | ✅ Implemented | Toggle heartbeat schedule enabled/disabled by name | `{ name, enabled }` |
| PATCH | `/api/heartbeat/{name}/schedule` | ✅ Implemented | Update cron schedule for a heartbeat by name | `{ name, schedule }` |
| GET | `/api/wallet` | ✅ Implemented | Wallet info (no mnemonic): current index, address, wordCount | `{ exists, currentIndex?, address?, defaultChain?, wordCount? }` |
| GET | `/api/wallet/address` | ✅ Implemented | Derive address for chain (optional index) | `{ chain, index, address }` |
| GET | `/api/wallet/identity-labels` | ✅ Implemented | Identity labels (HD index → name) from automaton.json | `{ identityLabels: Record<string, string> }` |
| PUT | `/api/wallet/identity-labels` | ✅ Implemented | Replace identity labels (auth required) | `{ identityLabels }` → `{ ok, identityLabels }` |
| PATCH | `/api/wallet/identity-labels` | ✅ Implemented | Update one or merge labels (auth required) | `{ index, label }` or `{ identityLabels }` → `{ ok, identityLabels }` |
| POST | `/api/wallet/rotate` | ✅ Implemented | Rotate HD account index (preview or confirm; auth required) | `{ currentIndex, targetIndex, currentAddresses, newAddresses, confirmed? }` |
| POST | `/api/wallet/clear-cache` | ✅ Implemented | Clear derived keys cache (auth required) | `{ ok, message }` |

### 2.3 Query Parameters

- **GET /api/status**: `?chain=<CAIP-2>` — Override chain for address resolution (e.g. `eip155:8453`).
- **GET /api/wallet/address**: `?chain=<CAIP-2>` (required) — Chain for address derivation. `?index=<N>` (optional) — HD index; 0 or omit = use wallet's current index.
- **GET /api/skills**: `?filter=all|enabled|disabled` — Filter by enabled state. `?trusted=all|trusted|untrusted` — Filter by trust (registry/builtin = trusted).
- **GET /api/skills/discovery**: `?q=<query>` — Search ClawHub by query. `?limit=N` — Max results (default 20). `?cursor=<token>` — Pagination for list (when no q).

### 2.4 POST /api/chat Request Body

```json
{ "message": "status" }
```

Supported messages: `status`, `help`, `帮助` — others return a generic "I don't understand" response.

### 2.5 PUT /api/config Request

- **Content-Type:** `application/json`
- **Body:** Full `automaton.json` content (JSON object matching `AutomatonConfig` schema).
- **Note:** Sends full config; partial updates overwrite missing fields with defaults. Restart required to apply changes.

### 2.6 GET /api/soul/config

Returns soul configuration: personality, system prompt, tone, and behavioral constraints. Shapes how the agent presents itself and responds.

**Response:**
```json
{
  "systemPrompt": "string",
  "personality": "string",
  "tone": "string",
  "behavioralConstraints": ["string", "..."],
  "systemPromptVersions": ["string", "..."]
}
```

`systemPromptVersions` — Last 30 system prompts (newest first), for history/rollback.

### 2.7 PUT /api/soul/config

Updates soul configuration. Accepts partial JSON; only provided fields are updated.

- **Content-Type:** `application/json`
- **Body (partial):**
```json
{
  "systemPrompt": "You are a helpful financial assistant...",
  "personality": "helpful, analytical, curious",
  "tone": "professional",
  "behavioralConstraints": ["Never disclose private keys", "Always verify before executing"]
}
```
- **Response:** `204 No Content`

### 2.8 POST /api/soul/enhance (authorized)

Turns casual words into a complete system prompt via LLM (soul enhancer). Requires `Authorization: Bearer <token>`.

- **Content-Type:** `application/json`
- **Body:**
```json
{
  "words": "helpful financial assistant, warm tone",
  "apply": false
}
```

- `words` — A few casual words describing the desired agent.
- `apply` — If `true`, saves the enhanced prompt to `automaton.json` and adds to version history (max 30). If `false`, returns preview only.

**Response:**
```json
{
  "systemPrompt": "You are a helpful financial assistant..."
}
```

### 2.9 POST /api/auth/verify Request

Verifies a signed message across multiple chains using `github.com/morpheum-labs/standards` components.

- **Content-Type:** `application/json`
- **Body:**
```json
{
  "chain": "ethereum|solana|bitcoin|morpheum",
  "message": "string",
  "signature": "string",
  "address": "string (optional for ethereum; required for solana/bitcoin)",
  "ecPubBytes": "base64 (required for morpheum)",
  "mldsaPubBytes": "base64 (required for morpheum)"
}
```

| Chain | Message format | Signature format |
|-------|----------------|-------------------|
| **ethereum** | Raw (EIP-191 personal_sign) | 0x-prefixed hex, 65 bytes |
| **solana** | Raw bytes | Base64 or base58, 64 bytes Ed25519 |
| **bitcoin** | Bitcoin Signed Message | Base64 |
| **morpheum** | Raw (SHA-256 hashed) | Base64, hybrid ECDSA+ML-DSA-44 |

**Response:** On success: `{ "valid": true, "address": "0x...", "token": "<JWT>" }` — the JWT is issued for user operations (pause, resume, chat, config) and should be sent as `Authorization: Bearer <token>` on write endpoints. On failure: `{ "valid": false, "error": "..." }`. JWT secret: `MONEYCLAW_JWT_SECRET` env var, or auto-generated at startup (tokens invalid on restart).

### 2.10 POST /api/auth/dev-bypass (dev only)

When `MONEYCLAW_DEV_BYPASS=1`, returns a JWT for address `0xdev` without wallet signing. For agent browser / automated testing.

- **Method:** POST
- **Body:** Optional JSON (ignored)
- **Response:** `{ "valid": true, "address": "0xdev", "token": "<JWT>" }`

To trigger from agent browser: `fetch("/api/auth/dev-bypass", { method: "POST" })` then store `response.token` in `localStorage.setItem("dashos:bearer", token)` and reload.

### 2.11 GET /api/reports

Returns metric reports from the `report_metrics` heartbeat task:

- **last_report**: JSON from KV `last_metrics_report` — `{ status, checkedAt, alerts?, error? }`
- **snapshots**: Recent rows from `metric_snapshots` — `[{ id, snapshot_at, metrics, alerts }]`

Metrics include `balance_cents`, `survival_tier`. Alerts indicate critical conditions (e.g. survival tier dead/critical).

### 2.12 Model List API

Model list configuration: add, remove, prioritize LLM providers; set API keys, model IDs, context limits, and cost caps.

**GET /api/models** — List configured models (API keys masked) and available providers.

**Response:**
```json
{
  "models": [
    {
      "id": "groq_llama-3.3-70b-versatile",
      "provider": "groq",
      "modelId": "llama-3.3-70b-versatile",
      "apiKeyMasked": "sk••••••••xyz",
      "contextLimit": 8192,
      "costCapCents": 500,
      "priority": 0,
      "enabled": true
    }
  ],
  "providers": [
    { "key": "openai", "displayName": "OpenAI", "local": false },
    { "key": "groq", "displayName": "Groq", "local": false }
  ]
}
```

**POST /api/models** — Add a model.

- **Body:** `{ "provider": "groq", "modelId": "llama-3.3-70b-versatile", "apiKey": "sk-...", "contextLimit": 8192, "costCapCents": 500, "enabled": true }`
- **Response:** `201 Created` + model object (apiKey masked)

**PATCH /api/models/{id}** — Update a model (partial). Fields: `apiKey`, `modelId`, `contextLimit`, `costCapCents`, `enabled`.

**DELETE /api/models/{id}** — Remove a model. Response: `204 No Content`.

**PUT /api/models/order** — Reorder models by priority.

- **Body:** `{ "ids": ["id1", "id2", "id3"] }` — order defines priority (first = highest)
- **Response:** `204 No Content`

### 2.13 Tools API

**GET /api/tools** — List registered tools. Requires `ToolsLister` in ServerConfig. Returns `{ tools: [{ name, description, enabled }] }`. Enabled state stored in KV `disabled_tools`.

**PATCH /api/tools/{name}** — Toggle tool enabled. Body: `{ "enabled": true|false }`. Response: `{ name, enabled }`.

### 2.14 Heartbeat Schedule API

Heartbeat schedules are cron jobs stored in the `heartbeat_schedule` DB table. Requires `state.Database` (or any DB implementing `HeartbeatScheduleAPI`).

**GET /api/heartbeat** — List all heartbeat schedule rows. Response:
```json
{
  "schedules": [
    {
      "name": "report_metrics",
      "schedule": "0 */6 * * *",
      "task": "report_metrics",
      "enabled": true,
      "tierMinimum": "dead",
      "lastRun": "2026-03-15T12:00:00Z",
      "nextRun": "",
      "leaseUntil": "",
      "leaseOwner": ""
    }
  ]
}
```

**PATCH /api/heartbeat/{name}** — Toggle enabled/disabled by name. Body: `{ "enabled": true|false }`. Response: `{ name, enabled }`. Returns `404` if schedule not found.

**PATCH /api/heartbeat/{name}/schedule** — Update cron schedule. Body: `{ "schedule": "0 */6 * * *" }`. Response: `{ name, schedule }`. Returns `404` if schedule not found.

### 2.15 Social Channels API

**GET /api/social** — List channels (Conway, Telegram, Discord, etc.) with status, config schema, and current values. Returns `{ channels: [{ name, displayName, enabled, ready, configFields, config }] }`.

**PATCH /api/social/{name}** — Toggle channel enabled. Body: `{ "enabled": true|false }`. Updates `socialChannels` in config.

**PUT /api/social/{name}/config** — Update channel config. Body: partial config object. Validates via `HealthCheck` before save; auto-enables on success. Response: `{ ok, validated?, enabled?, error? }`.

### 2.16 Tunnel API

Tunnel providers (bore, localtunnel, cloudflare, ngrok, tailscale, custom) expose local ports to the internet. Providers that require API keys: cloudflare (token), ngrok (authToken), tailscale (authKey).

**GET /api/tunnels** — List active tunnels. Response: `{ tunnels: [{ port, provider, public_url }] }`. Requires `TunnelManager` in ServerConfig.

**GET /api/tunnels/providers** — List provider names, schemas (fields per provider), and current config (secrets masked as `***`). Response:
```json
{
  "providers": ["bore", "localtunnel", "cloudflare", "ngrok", "tailscale", "custom"],
  "schemas": {
    "cloudflare": { "fields": [{ "name": "token", "type": "password", "required": true, "label": "Tunnel Token" }] },
    "ngrok": { "fields": [{ "name": "authToken", "type": "password", "required": true }, { "name": "domain", "type": "string", "required": false }] },
    "tailscale": { "fields": [{ "name": "authKey", "type": "password", "required": true }, { "name": "hostname", "type": "string", "required": false }, { "name": "funnel", "type": "boolean", "required": false }] }
  },
  "config": { "defaultProvider": "bore", "providers": { "cloudflare": { "enabled": true, "token": "***" } } }
}
```

**PUT /api/tunnels/providers/{name}** — Update provider config in automaton.json. Partial update; only provided fields are changed.

- **Content-Type:** `application/json`
- **Body (partial):** `{ "enabled"?, "token"?, "authToken"?, "authKey"?, "domain"?, "hostname"?, "funnel"?, "startCommand"?, "urlPattern"? }`
- **Response:** `{ "ok": true, "provider": "cloudflare" }`
- **Example:** `curl -X PUT .../api/tunnels/providers/cloudflare -d '{"token": "${CLOUDFLARE_TUNNEL_TOKEN}", "enabled": true}'`

**POST /api/tunnels/providers/{name}/restart** — Reload the provider from automaton.json. Stops all active tunnels, re-registers providers. For cloudflare/ngrok/tailscale, the required API key must be present in config; returns `400` otherwise. Requires `TunnelReloader` in ServerConfig.

- **Response:** `{ "ok": true, "provider": "cloudflare", "restarted": true }`
- **Errors:** `400` if provider requires API key but it is missing; `404` if TunnelReloader not configured.

### 2.17 Skills API

Skills are agent capabilities loaded from files (SKILL.md/SKILL.toml) or the ClawHub registry. The API supports list, CRUD, discovery, recommended skills, and activate/deactivate.

**Trusted vs untrusted:** `source` = `registry` or `builtin` → trusted; `source` = `installed` → untrusted (local path).

**GET /api/skills** — List installed skills.

- **Query:** `?filter=all|enabled|disabled` — Filter by enabled state.
- **Query:** `?trusted=all|trusted|untrusted` — Filter by trust.
- **Response:**
```json
{
  "skills": [
    {
      "name": "gmail-secretary",
      "description": "Gmail triage assistant...",
      "source": "registry",
      "path": "/Users/me/.automaton/skills/gmail-secretary",
      "enabled": true,
      "trusted": true,
      "auto_activate": 1
    }
  ]
}
```

**GET /api/skills/{name}** — Get single skill. Returns `404` if not found.

**GET /api/skills/discovery** — Discover skills from ClawHub.

- **Query:** `?q=<query>` — Search by text (vector search).
- **Query:** `?limit=N` — Max results (default 20, max 100).
- **Query:** `?cursor=<token>` — Pagination when listing (no `q`).
- **Response (search):** `{ "results": [{ "slug", "displayName", "summary", "version", "score" }] }`
- **Response (list):** `{ "items": [...], "nextCursor": "..." }`

**GET /api/skills/recommended** — Curated recommended skills from ClawHub.

- **Response:** `{ "recommended": [{ "slug", "displayName", "summary", "version", "installed" }] }`

**POST /api/skills** — Install a skill.

- **Content-Type:** `application/json`
- **Body (ClawHub):**
```json
{
  "source": "clawhub",
  "id": "gmail-secretary",
  "version": "1.0.20",
  "name": "Gmail Secretary",
  "description": "Optional override"
}
```
- **Body (local path):**
```json
{
  "name": "my-skill",
  "path": "/path/to/skill-dir",
  "description": "Optional"
}
```
- **Response:** `201 Created` + `{ "name", "source", "path", "enabled" }`
- **Errors:** `400` for invalid body or install failure; `503` if skills API not available (DB does not implement SkillsAPIStore).

**PATCH /api/skills/{name}** — Update skill (partial).

- **Body:** `{ "enabled"?: bool, "description"?: string, "instructions"?: string }`
- **Response:** `{ "name", "description", "enabled" }`

**DELETE /api/skills/{name}** — Remove skill. Response: `204 No Content`.

**PATCH /api/skills/{name}/activate** — Enable skill. Response: `{ "name", "enabled": true }`.

**PATCH /api/skills/{name}/deactivate** — Disable skill. Response: `{ "name", "enabled": false }`.

**Config:** Registry URL and timeout from `skills.registry` in automaton.json, or `SkillsConfigGetter` in ServerConfig. Default: `https://clawhub.ai`, 30s timeout.

### 2.18 Wallet API

Mnemonic wallet management: multi-chain address derivation, HD index rotation, cache clear. **Never exposes mnemonic or private keys.**

**GET /api/wallet** — Wallet info (no secrets).

- **Response (exists):** `{ exists: true, currentIndex, address, defaultChain, wordCount }`
- **Response (no wallet):** `{ exists: false, error: "no wallet: run 'moneyclaw init' first" }`

**GET /api/wallet/address** — Derive address for a chain.

- **Query:** `chain` (required, CAIP-2 e.g. `eip155:8453`)
- **Query:** `index` (optional; 0 or omit = use wallet's current HD index)
- **Response:** `{ chain, index, address }`
- **Errors:** `400` if chain missing or invalid; `400` if no wallet or derivation fails

**GET /api/wallet/identity-labels** — Identity labels (HD index → friendly name) from automaton.json.

- **Response:** `{ identityLabels: { "0": "Main", "1": "Trading", ... } }`

**PUT /api/wallet/identity-labels** — Replace all identity labels. **Requires `Authorization: Bearer <token>`.**
- **Body:** `{ "identityLabels": { "0": "Main", "1": "Trading", ... } }`
- **Response:** `{ ok: true, identityLabels: { ... } }`

**PATCH /api/wallet/identity-labels** — Update one label or merge. **Requires `Authorization: Bearer <token>`.**
- **Body (single):** `{ "index": 1, "label": "Trading" }` — empty label removes entry
- **Body (merge):** `{ "identityLabels": { "1": "Trading" } }` — merge into existing
- **Response:** `{ ok: true, identityLabels: { ... } }`

**POST /api/wallet/rotate** — Rotate HD account index. **Requires `Authorization: Bearer <token>`.**

- **Content-Type:** `application/json`
- **Body:** `{ "toIndex": N, "preview"?: bool, "confirm"?: bool }`
  - `toIndex` — Target HD account index
  - `preview` — If true, return current/new addresses without writing
  - `confirm` — If true, write new index to wallet.json and clear cache
- **Response:** `{ currentIndex, targetIndex, currentAddresses: { chain: address }, newAddresses: { chain: address }, preview, confirmed, message? }`
- **Chains shown:** defaultChain + chainProviders from config
- **Note:** Does not sweep funds; operator must migrate balances manually

**POST /api/wallet/clear-cache** — Clear derived keys cache. **Requires `Authorization: Bearer <token>`.**

- **Response:** `{ ok: true, message: "derived keys cache cleared" }`

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
| **Web Dashboard** | `/`, `/static/*`, `GET /api/status`, `GET /api/strategies`, `GET /api/history`, `GET /api/cost`, `GET /api/risk`, `POST /api/pause`, `POST /api/resume`, `POST /api/chat`, `GET /api/config`, `PUT /api/config`, `GET /api/soul/config`, `PUT /api/soul/config`, `GET /api/tools`, `PATCH /api/tools/{name}`, `GET /api/social`, `PATCH /api/social/{name}`, `PUT /api/social/{name}/config`, `GET /api/tunnels`, `GET /api/tunnels/providers`, `PUT /api/tunnels/providers/{name}`, `POST /api/tunnels/providers/{name}/restart`, `GET /api/models`, `POST /api/models`, `PATCH /api/models/{id}`, `DELETE /api/models/{id}`, `PUT /api/models/order`, `GET /api/skills`, `GET /api/skills/discovery`, `GET /api/skills/recommended`, `GET /api/skills/{name}`, `POST /api/skills`, `PATCH /api/skills/{name}`, `DELETE /api/skills/{name}`, `PATCH /api/skills/{name}/activate`, `PATCH /api/skills/{name}/deactivate`, `GET /api/heartbeat`, `PATCH /api/heartbeat/{name}`, `PATCH /api/heartbeat/{name}/schedule`, `GET /api/wallet`, `GET /api/wallet/address`, `POST /api/wallet/rotate`, `POST /api/wallet/clear-cache`, `POST /api/auth/verify`, `GET /api/reports` |
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
| `POST /api/chat` | ⚠️ Keyword-only | status/help/帮助 |
| `GET /api/config` | ✅ Full | Reads automaton.json; returns `{ content: string }` |
| `PUT /api/config` | ✅ Full | Accepts full JSON config; writes to automaton.json |
| `GET /api/soul/config` | ✅ Full | Soul config (personality, system prompt, tone, behavioral constraints) from config |
| `PUT /api/soul/config` | ✅ Full | Partial merge; writes to automaton.json `soul` section |
| `GET /api/tools` | ✅ Full | Lists tools from ToolsLister; enabled state from KV `disabled_tools` |
| `PATCH /api/tools/{name}` | ✅ Full | Toggle enabled; persists to KV |
| `GET /api/social` | ✅ Full | Lists channels via social factory; status, config schema, values |
| `PATCH /api/social/{name}` | ✅ Full | Toggle enabled; updates socialChannels in config |
| `PUT /api/social/{name}/config` | ✅ Full | Channel config; validates via HealthCheck before save |
| `GET /api/models` | ✅ Full | Model list from config; API keys masked |
| `POST /api/models` | ✅ Full | Add model; generates id |
| `PATCH /api/models/{id}` | ✅ Full | Update model fields |
| `DELETE /api/models/{id}` | ✅ Full | Remove model |
| `PUT /api/models/order` | ✅ Full | Reorder by priority |
| `POST /api/auth/verify` | ✅ Full | Verifies message signatures (Ethereum, Solana, Bitcoin, Morpheum) via standards package |
| `GET /api/reports` | ✅ Full | Last metrics report (KV) + recent metric_snapshots from report_metrics task |
| `GET /api/tunnels` | ✅ Full | Active tunnels from TunnelManager.Status() |
| `GET /api/tunnels/providers` | ✅ Full | Provider schemas + config (secrets masked); loads from config |
| `PUT /api/tunnels/providers/{name}` | ✅ Full | Update provider config in automaton.json; partial merge |
| `POST /api/tunnels/providers/{name}/restart` | ✅ Full | Reload providers via TunnelReloader; validates API key for cloudflare/ngrok/tailscale |
| `GET /api/skills` | ✅ Full | List installed skills; filter by enabled/trusted |
| `GET /api/skills/discovery` | ✅ Full | Search or list ClawHub registry |
| `GET /api/skills/recommended` | ✅ Full | Curated recommended skills |
| `GET /api/skills/{name}` | ✅ Full | Get single skill |
| `POST /api/skills` | ✅ Full | Install from ClawHub or local path |
| `PATCH /api/skills/{name}` | ✅ Full | Update enabled, description, instructions |
| `DELETE /api/skills/{name}` | ✅ Full | Remove skill |
| `PATCH /api/skills/{name}/activate` | ✅ Full | Enable skill |
| `PATCH /api/skills/{name}/deactivate` | ✅ Full | Disable skill |
| `GET /api/heartbeat` | ✅ Full | List heartbeat_schedule rows from DB |
| `PATCH /api/heartbeat/{name}` | ✅ Full | Toggle heartbeat enabled by name |
| `PATCH /api/heartbeat/{name}/schedule` | ✅ Full | Update cron schedule by name |
| `GET /api/wallet` | ✅ Full | Wallet info (no mnemonic); current index, address, wordCount |
| `GET /api/wallet/address` | ✅ Full | Derive address for chain (query: chain, index?) |
| `GET /api/wallet/identity-labels` | ✅ Full | Identity labels (index → name) from automaton.json |
| `PUT /api/wallet/identity-labels` | ✅ Full | Replace identity labels (JWT required) |
| `PATCH /api/wallet/identity-labels` | ✅ Full | Update/merge identity labels (JWT required) |
| `POST /api/wallet/rotate` | ✅ Full | Rotate HD index (preview/confirm; JWT required) |
| `POST /api/wallet/clear-cache` | ✅ Full | Clear derived keys cache (JWT required) |

**Auth:** None. Dashboard is unauthenticated. Wallet rotate and clear-cache require JWT (via `POST /api/auth/verify`). Anyone reaching the URL has full access. Write endpoints can optionally require auth via `POST /api/auth/verify` flow.

### 8.2 Implementable in internal/web

Endpoints that can be added using existing data sources. Requires extending `Server` with optional dependencies (type-assert `WebDB` to `*state.Database`, pass `Tools`, `TunnelManager`, `Config`, etc.).

| Endpoint | Data Source | Effort | Description |
|----------|-------------|--------|-------------|
| `GET /api/history` | `state.GetRecentTurns(limit)` | Low | Turn history: id, timestamp, state, input, thinking, tool_calls, cost_cents. Extend WebDB or type-assert to `*state.Database`. |
| `GET /api/soul` | `DB.GetKV("soul_content")` | Low | Read soul *document* from KV (self-authored identity). Distinct from `GET /api/soul/config` which returns soul *configuration* (personality, tone, constraints). Same pattern as `view_soul` tool. |
| `GET /api/memory` | `state.GetSemanticMemory`, `GetProceduralMemory`, KV goals/facts | Medium | Facts, goals, procedures. Requires schema v13 (5-tier) + KV keys like `goal:`, `procedure:`. |
| `GET /api/health` | — | Trivial | Liveness: return `200 OK` or `{"ok": true}`. |
| `GET /api/heartbeat`, `PATCH /api/heartbeat/{name}`, `PATCH /api/heartbeat/{name}/schedule` | — | — | ✅ Implemented. See §2.14. |
| `GET /api/tunnels`, `GET /api/tunnels/providers`, `PUT /api/tunnels/providers/{name}`, `POST /api/tunnels/providers/{name}/restart` | — | — | ✅ Implemented. See §2.16. |
| `POST /api/chat` (enhance) | Inference client or agent loop | High | Wire to real LLM or forward to agent. Frontend must call `POST /api/chat` with body `{ "message": "..." }`. |

### 8.3 Frontend Gaps

- **History:** No UI for `/api/history`; add a panel when endpoint returns real data.

---

## 9. References

- [ts-go-alignment.md](design/ts-go-alignment.md) — Web API alignment with TypeScript reference
- [child-runtime-protocol.md](design/child-runtime-protocol.md) — Conway sandbox operations
- [wallet-identity-architecture.md](design/wallet-identity-architecture.md) — SIWE provisioning flow
