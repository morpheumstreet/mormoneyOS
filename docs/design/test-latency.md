# Test-Latency API Design

**Date:** 2026-03-16  
**Purpose:** API endpoint for local models to measure response wait time. Auth-required, rate-limited, configurable cooldown.

---

## 1. Requirements

| Requirement | Implementation |
|-------------|----------------|
| Bearer token required | `validateJWT(r)`; `401 Unauthorized` if missing/invalid |
| 120s cooldown per model | Rate limiter key = (user, provider, url, model); testing model A does not block testing model B |
| Configurable cooldown | `testLatencyCooldownSeconds` in `automaton.json`, default 120 |
| Modularity | Separate interfaces and single responsibilities |

---

## 2. API

**Endpoint:** `POST /api/models/test-latency`

**Query params:** `provider`, `url`, `model` (required)

**Headers:** `Authorization: Bearer <jwt>` (required)

**Success (200):**
```json
{"latencyMs": 123}
```

**Rate limited (429):**
```json
{"error": "rate limited", "retryAfter": 95}
```
`Retry-After` header also set.

**Probe failure (502):**
```json
{"error": "request: connection refused"}
```

---

## 3. Config

**`automaton.json`:**
```json
{
  "testLatencyCooldownSeconds": 120
}
```
Default: 120 when omitted.

---

## 4. Component Responsibilities

| Component | Responsibility |
|-----------|----------------|
| Auth | Validate JWT; reject invalid requests |
| Rate limit | Enforce cooldown per (user, provider, url, model) |
| Probe | Measure latency; return ms or error |
| Handler | Orchestrate auth → rate limit → probe; return JSON |

---

## 5. Components

| Component | Location |
|-----------|----------|
| `TestLatencyCooldownSeconds` | `internal/types/types.go` |
| Config merge | `internal/config/config.go` |
| `RateLimiter` + `MemoryRateLimiter` | `internal/ratelimit/ratelimit.go` |
| `LatencyProber` + `DefaultLatencyProber` | `internal/inference/latency_prober.go` |
| Handler | `internal/web/server.go` (`handleAPIModelsTestLatency`) |
| Wiring | `cmd/run.go` |
| Frontend API | `dashos/src/lib/api.ts` (`postModelsTestLatency`) |

---

## 6. Supported Providers

Local providers only: `ollama`, `localai`, `llamacpp`, `lmstudio`, `vllm`, `janai`, `g4f`.

- **Ollama:** `POST /api/generate` (native)
- **Others:** `POST /v1/chat/completions` (OpenAI-compatible)
