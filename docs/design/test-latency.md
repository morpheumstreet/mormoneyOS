# Test-Latency API Design

**Date:** 2026-03-16  
**Purpose:** API endpoint for local models to measure response wait time. Auth-required, rate-limited, configurable cooldown. Follows SOLID principles.

---

## 1. Requirements

| Requirement | Implementation |
|-------------|----------------|
| Bearer token required | `validateJWT(r)`; `401 Unauthorized` if missing/invalid |
| 120s cooldown per model | Rate limiter key = (user, provider, url, model); testing model A does not block testing model B |
| Configurable cooldown | `testLatencyCooldownSeconds` in `automaton.json`, default 120 |
| SOLID | Separate interfaces and single responsibilities |

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

## 4. SOLID Design

| Principle | Application |
|-----------|-------------|
| **S**ingle Responsibility | Auth, rate limit, probe, handler each have one job |
| **O**pen/Closed | New rate-limit or probe strategies without changing handler |
| **L**iskov Substitution | Any `RateLimiter` or `LatencyProber` implementation works |
| **I**nterface Segregation | `RateLimiter.Allow()`, `LatencyProber.Probe()` minimal surfaces |
| **D**ependency Inversion | Handler depends on interfaces; implementations injected |

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
