# Inference Client Hot-Reload Design

**Date:** 2026-03-16  
**Purpose:** Apply inference model/provider changes from the web UI without restarting the process. Aligns with the existing `TunnelReloader` pattern.

---

## 1. Context

### 1.1 Current State

| Component | Behavior |
|-----------|----------|
| `cmd/run.go` | Creates `infClient := inference.NewClientFromConfig(cfg)` once at startup. |
| Agent loop | Receives `infClient`; uses it for `Chat`, `GetDefaultModel`, `SetLowComputeMode`. |
| Web server | Receives `ChatClient: infClient`; uses it for Agent Comm Link (`POST /api/chat`). |
| `PUT /api/config` | Saves config to `automaton.json`; does not update the running inference client. |

### 1.2 Desired State

- Config changes (provider, inferenceModel, models, API keys) applied via web UI take effect immediately.
- No process restart required.
- Same pattern as tunnel reload: explicit reload, no magic.

---

## 2. Architecture

### 2.1 Holder + Live Client

```
┌─────────────────────────────────────────────────────────────────┐
│                    InferenceClientHolder                         │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  mu sync.RWMutex                                             ││
│  │  client inference.Client                                    ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  Client() inference.Client   ← returns current (thread-safe)     │
│  Reload(cfg)                 ← swap client atomically            │
└─────────────────────────────────────────────────────────────────┘
         │
         │ wraps
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  LiveInferenceClient (implements inference.Client)               │
│  - Chat()        → holder.Client().Chat(...)                     │
│  - GetDefaultModel() → holder.Client().GetDefaultModel()         │
│  - SetLowComputeMode() → holder.Client().SetLowComputeMode()     │
└─────────────────────────────────────────────────────────────────┘
         │
         │ injected into
         ▼
┌──────────────────────┐    ┌──────────────────────┐
│   Agent Loop         │    │   Web Server          │
│   (inference.Client) │    │   (ChatClient)        │
└──────────────────────┘    └──────────────────────┘
```

### 2.2 Flow

1. **Startup:** Create holder with initial config → create live client → pass live client to agent loop and web server.
2. **Config save:** `PUT /api/config` → save to disk → call `holder.Reload(cfg)`.
3. **Next inference call:** Agent loop or chat handler calls `Chat()` → live client delegates to `holder.Client()` → gets new client.

---

## 3. Implementation

### 3.1 New Types (`internal/inference/holder.go`)

```go
package inference

import (
	"context"
	"sync"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// InferenceClientHolder holds the current inference client and supports atomic reload.
// Thread-safe; safe for concurrent reads and reloads.
type InferenceClientHolder struct {
	mu     sync.RWMutex
	client Client
}

// NewInferenceClientHolder creates a holder with the client from cfg.
func NewInferenceClientHolder(cfg *types.AutomatonConfig) *InferenceClientHolder {
	h := &InferenceClientHolder{}
	h.Reload(cfg)
	return h
}

// Client returns the current inference client. Safe for concurrent use.
func (h *InferenceClientHolder) Client() Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.client
}

// Reload creates a new client from cfg and swaps it atomically.
func (h *InferenceClientHolder) Reload(cfg *types.AutomatonConfig) {
	newClient := NewClientFromConfig(cfg)
	h.mu.Lock()
	defer h.mu.Unlock()
	h.client = newClient
}

// LiveClient returns an inference.Client that always delegates to the holder's current client.
// Use this when injecting into the agent loop and web server.
func (h *InferenceClientHolder) LiveClient() Client {
	return &liveInferenceClient{holder: h}
}

type liveInferenceClient struct {
	holder *InferenceClientHolder
}

func (c *liveInferenceClient) Chat(ctx context.Context, messages []ChatMessage, opts *InferenceOptions) (*InferenceResponse, error) {
	return c.holder.Client().Chat(ctx, messages, opts)
}

func (c *liveInferenceClient) GetDefaultModel() string {
	return c.holder.Client().GetDefaultModel()
}

func (c *liveInferenceClient) SetLowComputeMode(enabled bool) {
	c.holder.Client().SetLowComputeMode(enabled)
}
```

### 3.2 ServerConfig Extension (`internal/web/server.go`)

```go
// ServerConfig
type ServerConfig struct {
	// ... existing fields ...
	ChatClient       inference.Client
	TunnelReloader   func(cfg *types.TunnelConfig)
	InferenceReloader func(cfg *types.AutomatonConfig)  // NEW: when set, config save triggers inference reload
}
```

### 3.3 Config Put Handler Change

```go
// handleAPIConfigPut: after config.Save(&cfg)
if s.Cfg != nil && s.Cfg.InferenceReloader != nil {
	s.Cfg.InferenceReloader(&cfg)
}
```

### 3.4 cmd/run.go Wiring

```go
// Create holder instead of raw client
infHolder := inference.NewInferenceClientHolder(cfg)
infClient := infHolder.LiveClient()

// Agent loop
loop := agent.NewLoopWithOptions(agent.LoopOptions{
	Inference: infClient,
	// ...
})

// Web server
webSrv := web.NewServer(webAddr, webState, db, &web.ServerConfig{
	ChatClient:        infClient,
	InferenceReloader: func(cfg *types.AutomatonConfig) { infHolder.Reload(cfg) },
	// ...
})
```

---

## 4. API Surface

### 4.1 No New HTTP Endpoint

Reload is triggered implicitly when config is saved via `PUT /api/config`. This is consistent with the expectation: "change model in UI → save → apply." No separate "Apply" or "Reload inference" button needed if the UX is "save config = apply."

### 4.2 Optional: Explicit Reload Endpoint

If product requires an explicit "Apply model changes" action without full config save:

```
POST /api/inference/reload
```

- Loads config from disk (`config.Load()`)
- Calls `InferenceReloader(cfg)` if configured
- Returns `204 No Content` or `404` if reloader not set

This is optional; the design works with config-save-only.

---

## 5. Model Source: Legacy vs cfg.Models

### 5.1 Current Behavior

`NewClientFromConfig` uses:
- `cfg.Provider` (explicit)
- `cfg.InferenceModel` (explicit)
- `providerResolutionOrder` (auto-detect from API keys)
- Fallback: ChatJimmy

### 5.2 Design Choice

**Phase 1 (this design):** Keep `NewClientFromConfig`. Reload uses the same logic. Config UI edits to `provider`, `inferenceModel`, and API keys apply on save.

**Phase 2 (future):** If we want the primary inference client to come from `cfg.Models` (first enabled by priority), we could add:

```go
func (h *InferenceClientHolder) Reload(cfg *types.AutomatonConfig) {
	newClient := BestEnhanceClient(cfg)  // or NewClientFromConfig(cfg) for legacy
	// ...
}
```

Configurable via a flag or config key. Out of scope for this design.

---

## 6. Edge Cases

| Case | Handling |
|------|----------|
| Reload during active Chat() | In-flight request uses old client; next request uses new. No coordination needed. |
| Config load error on reload | Keep current client; log error. Do not swap in nil. |
| Nil cfg passed to Reload | No-op or panic; caller (config put handler) always has valid cfg. |
| Agent loop mid-turn | Current turn uses client at start of turn; next turn gets new client. Acceptable. |

---

## 7. Testing

| Test | Description |
|------|-------------|
| `TestInferenceClientHolder_Reload` | Create holder, call Client(), Reload with different provider, assert Client() returns new client. |
| `TestLiveInferenceClient_Delegates` | Create holder + live client, Reload, verify Chat() uses new client. |
| `TestInferenceReloader_OnConfigPut` | Mock server with InferenceReloader; PUT config; assert reloader called with saved cfg. |

---

## 8. Summary

| Change | Location |
|--------|----------|
| `InferenceClientHolder` + `LiveClient` | `internal/inference/holder.go` (new) |
| `InferenceReloader` in ServerConfig | `internal/web/server.go` |
| Call reloader after config save | `handleAPIConfigPut` |
| Use holder in run | `cmd/run.go` |

**Result:** Config changes from the web UI apply to inference (agent loop + Agent Comm Link) immediately, without restart. Design mirrors `TunnelReloader`; minimal surface; thread-safe.
