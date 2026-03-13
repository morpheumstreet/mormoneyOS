# mormclaw Provider Design → mormoneyOS Borrow

**Date:** 2025-03-13  
**Purpose:** Capture mormclaw (mormOS) AI provider architecture and recommend patterns to borrow into mormoneyOS. Optimized for **clean**, **DRY**, and **SOLID** design.

---

## 0. Design Principles (Clean, DRY, SOLID)

### 0.1 Clean

| Principle | Application |
|-----------|-------------|
| **Single responsibility** | Each provider does one thing: call its API and return normalized `InferenceResponse`. No business logic, no routing. |
| **Clear boundaries** | Provider = transport + auth. Agent loop = orchestration. Factory = construction. No cross-cutting concerns. |
| **Explicit over implicit** | Auth style, base URL, model — all passed explicitly. No magic defaults hidden in provider internals. |

### 0.2 DRY

| Principle | Application |
|-----------|-------------|
| **One OpenAI-compatible implementation** | All providers using `/v1/chat/completions` share a single `OpenAICompatibleClient`. No per-provider request/response code. |
| **Shared types** | `ChatMessage`, `InferenceResponse`, `ToolCall` live in `client.go`; all providers consume them. |
| **Shared HTTP + auth** | One `buildRequest` / `setAuthHeader` path parameterized by `AuthStyle`. |
| **Provider descriptors, not code** | New OpenAI-compatible provider = registry entry (name, baseURL, authStyle, configKey). No new file. |

### 0.3 SOLID

| Principle | Application |
|-----------|-------------|
| **S**ingle Responsibility | `Client` = inference only. Factory = construction. Registry = metadata. |
| **O**pen/Closed | Add providers by registering, not by editing factory switch. |
| **L**iskov Substitution | Any `Client` implementation works in agent loop; no provider-specific branches. |
| **I**nterface Segregation | `Client` has minimal surface: `Chat`, `GetDefaultModel`, `SetLowComputeMode`. No fat interface. |
| **D**ependency Inversion | Agent loop depends on `inference.Client` interface; factory injects concrete implementation. |

### 0.4 Provider Taxonomy (DRY)

```
┌─────────────────────────────────────────────────────────────────┐
│                    inference.Client (interface)                   │
└─────────────────────────────────────────────────────────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────┬───────────────────────────┐
        │                           │                           │                           │
        ▼                           ▼                           ▼                           ▼
┌───────────────┐       ┌───────────────────────┐       ┌───────────────┐       ┌───────────────┐
│ StubClient    │       │ OpenAICompatibleClient │       │ AnthropicClient│       │ ChatJimmyClient│
│ (no network)  │       │ (one impl, N providers)│       │ (custom API)   │       │ (chatjimmy.ai) │
└───────────────┘       └───────────────────────┘       └───────────────┘       └───────────────┘
                                 │
                    ┌────────────┼────────────┐
                    │            │            │
              OpenAI, Conway, Ollama, Groq, Mistral, DeepSeek, ...
              (differ only by: baseURL, authStyle, configKey)
```

---

## 1. mormclaw Provider Architecture (Reference)

### 1.1 Core Components

| Component | Path | Role |
|-----------|------|------|
| **Provider trait** | `src/providers/traits.rs` | Single interface: `chat`, `chat_with_system`, `chat_with_history`, `chat_with_tools` |
| **ProviderCapabilities** | `traits.rs` | `native_tool_calling`, `vision` — enables adaptation |
| **ToolsPayload** | `traits.rs` | Provider-specific formats: `Gemini`, `Anthropic`, `OpenAI`, `PromptGuided` |
| **NormalizedStopReason** | `traits.rs` | Provider-agnostic stop reasons (EndTurn, ToolCall, MaxTokens, etc.) |
| **OpenAiCompatibleProvider** | `src/providers/compatible.rs` | Generic client for any OpenAI-compatible `/v1/chat/completions` API |
| **Factory** | `src/providers/mod.rs` | `create_provider(name, api_key)` → `Box<dyn Provider>` by string key |
| **RouterProvider** | `src/providers/router.rs` | `hint:reasoning` → routes to different provider+model |
| **ReliableProvider** | `src/providers/reliable.rs` | Fallback chain across providers |

### 1.2 Provider Factory Pattern

```rust
// Single entry point by canonical key
create_provider(name: &str, api_key: Option<&str>) -> Result<Box<dyn Provider>>
create_provider_with_url(name, api_key, api_url) -> Result<Box<dyn Provider>>
create_provider_with_options(name, api_key, options) -> Result<Box<dyn Provider>>
```

**Provider keys** (examples): `openai`, `anthropic`, `ollama`, `openrouter`, `gemini`, `venice`, `groq`, `mistral`, `deepseek`, `xai`, `bedrock`, `glm`, `moonshot`, `qwen`, `chatjimmy`, etc.

**Alias resolution:** `google` → `gemini`, `grok` → `xai`, `together` → `together-ai`, `chatjimmy-cli` → `chatjimmy`, etc.

### 1.3 OpenAiCompatibleProvider (Key Borrow)

Most providers use the same `/v1/chat/completions` API. mormclaw uses one generic implementation:

```rust
OpenAiCompatibleProvider::new(name, base_url, credential, AuthStyle)
```

**AuthStyle:**
- `Bearer` — `Authorization: Bearer <key>`
- `XApiKey` — `x-api-key: <key>` (some Chinese providers)
- `Custom(String)` — custom header name

**Used for:** Venice, Vercel, Cloudflare, Moonshot, Groq, Mistral, xAI, DeepSeek, Together, Fireworks, GLM, MiniMax, Qwen, SiliconFlow, StepFun, Hunyuan, Qianfan, Doubao, etc.

### 1.4 Custom Implementations (Non-OpenAI API)

- **Anthropic** — Messages API (different request/response shape)
- **Gemini** — functionDeclarations format
- **Bedrock** — AWS SDK
- **OpenRouter** — multi-provider gateway
- **Ollama** — local, no auth by default
- **ChatJimmy** — chatjimmy.ai (Taalas HC1 inference). Custom `/api/chat` API with `chat_options`, `selected_model`, `system_prompt`; no auth. Default model: `llama3.1-8B`. API reference: [chatjimmy-cli](https://github.com/kichichifightclubx/chatjimmy-cli/blob/main/docs/02-api-reference.md)

### 1.5 RouterProvider (hint-based routing)

```
model: "hint:reasoning" → Route { provider: "smart", model: "claude-opus" }
model: "hint:fast"      → Route { provider: "fast", model: "llama-3-70b" }
model: "gpt-4o"         → default provider with model as-is
```

### 1.6 list_providers()

Display-only: `ProviderInfo { name, display_name, aliases, local }` — separate from factory for UI/CLI.

---

## 2. mormoneyOS Current State

| Component | Path | Role |
|-----------|------|------|
| **Client interface** | `internal/inference/client.go` | `Chat(ctx, messages, opts)` |
| **OpenAIClient** | `internal/inference/openai.go` | OpenAI-compatible (OpenAI, Conway, Ollama) |
| **StubClient** | `internal/inference/stub.go` | No-op when no keys |
| **NewClientFromConfig** | `internal/inference/factory.go` | Priority: OpenAI > Conway > Stub |

**Config fields:** `openaiApiKey`, `anthropicApiKey`, `conwayApiUrl`, `conwayApiKey`, `inferenceModel`, `maxTokensPerTurn`

---

## 3. Optimized Design (Clean, DRY, SOLID)

### 3.1 Single OpenAI-Compatible Implementation (DRY)

**One struct, one Chat implementation.** All OpenAI-compatible providers use it.

```go
// compatible.go
type AuthStyle int
const ( AuthBearer AuthStyle = iota; AuthXApiKey )

type OpenAICompatibleClient struct {
    Name      string    // for logging
    BaseURL   string
    APIKey    string
    AuthStyle AuthStyle
    Model     string
    MaxTokens int
    HTTP      *http.Client
}

func NewOpenAICompatibleClient(name, baseURL, apiKey string, auth AuthStyle, model string, maxTokens int) *OpenAICompatibleClient
```

**Provider-specific behavior = data, not code.** No `openai.go`, `groq.go`, `mistral.go` — only a registry.

### 3.2 Provider Registry (Open/Closed)

**Add providers without editing factory logic.** Registry is the single source of truth.

```go
// registry.go
type ProviderSpec struct {
    Key             string    // "openai", "groq", "conway", ...
    DisplayName     string
    BaseURL         string    // default; overridden by BaseURLConfigKey when set
    BaseURLConfigKey string   // e.g. "ConwayAPIURL" — read from config when non-empty
    AuthStyle       AuthStyle
    APIKeyConfigKey string    // e.g. "OpenAIAPIKey" — config field for API key
    Local           bool     // ollama = true, no key required
}

var registry = []ProviderSpec{
    {"openai", "OpenAI", "https://api.openai.com", "", AuthBearer, "OpenAIAPIKey", false},
    {"groq", "Groq", "https://api.groq.com/openai/v1", "", AuthBearer, "GroqAPIKey", false},
    {"conway", "Conway", "", "ConwayAPIURL", AuthXApiKey, "ConwayAPIKey", false},
    {"ollama", "Ollama", "http://localhost:11434", "", AuthBearer, "", true},
    // ...
}
```

Factory: `lookup(provider) → spec → resolveBaseURL(spec, cfg) → NewOpenAICompatibleClient(...)`. No provider-specific branches in factory.

### 3.3 Factory (Single Responsibility)

**Factory only constructs.** No routing, no fallback, no business logic.

```go
// factory.go
func NewClientFromConfig(cfg *types.AutomatonConfig) Client {
    provider := cfg.Provider
    if provider == "" {
        provider = resolveProviderFromKeys(cfg) // backward compat
    }
    spec := LookupProvider(provider)
    if spec == nil {
        return NewStubClient(cfg.InferenceModel)
    }
    key := getConfigValue(cfg, spec.APIKeyConfigKey)
    if key == "" && !spec.Local {
        return NewStubClient(cfg.InferenceModel)
    }
    baseURL := resolveBaseURL(spec, cfg) // spec.BaseURL or cfg[spec.BaseURLConfigKey]
    return NewOpenAICompatibleClient(spec.DisplayName, baseURL, key, spec.AuthStyle, cfg.InferenceModel, cfg.MaxTokensPerTurn)
}
```

### 3.4 Custom Providers (When Necessary)

**Only implement custom client when API shape differs.** Anthropic, OpenRouter, Bedrock = separate files. OpenAI-compatible = registry entry.

| Provider | Approach | Reason |
|----------|----------|--------|
| OpenAI, Groq, Mistral, DeepSeek, Conway, Ollama, ... | Registry + OpenAICompatibleClient | Same API |
| Anthropic | `anthropic.go` | Messages API, different schema |
| OpenRouter | `openrouter.go` | Multi-provider gateway |
| Bedrock | `bedrock.go` | AWS SDK |
| ChatJimmy | `chatjimmy.go` | chatjimmy.ai `/api/chat` API; no auth; custom request shape |

### 3.5 Interface Segregation

**Client stays minimal.** No provider-specific methods.

```go
type Client interface {
    Chat(ctx context.Context, messages []ChatMessage, opts *InferenceOptions) (*InferenceResponse, error)
    GetDefaultModel() string
    SetLowComputeMode(enabled bool)
}
```

Capabilities (vision, streaming) = future optional interfaces or opts, not now.

---

## 4. Implementation Order

1. **Rename + refactor** — `OpenAIClient` → `OpenAICompatibleClient`; add `AuthStyle`; move to `compatible.go`.
2. **Registry** — Add `registry.go` with `ProviderSpec` and initial entries (openai, conway, ollama).
3. **Factory** — Refactor `NewClientFromConfig` to use registry lookup.
4. **Add providers** — Groq, Mistral, DeepSeek = new registry entries + config keys.
5. **Anthropic** — Add `anthropic.go` when needed; factory branches on `spec.CustomImpl`.
6. **ChatJimmy** — Add `chatjimmy.go` for chatjimmy.ai (Taalas HC1, no auth); optional `chatjimmyApiUrl` config.

---

## 5. Config Shape

```yaml
# Primary
provider: "openai"       # explicit; or omit for backward-compat auto-detect
inferenceModel: "gpt-4o-mini"
maxTokensPerTurn: 4096

# Per-provider keys (factory reads based on provider)
openaiApiKey: "..."
anthropicApiKey: "..."
conwayApiUrl: "..."     # Conway needs URL + key
conwayApiKey: "..."
groqApiKey: "..."       # when provider: groq
chatjimmyApiUrl: "..."  # optional; default https://chatjimmy.ai (no key required)
# ...
```

**Backward compat:** If `provider` empty → `resolveProviderFromKeys(cfg)` (OpenAI > Conway > Stub).

---

## 6. File Layout (Optimized)

```
internal/inference/
  client.go       # Client interface, ChatMessage, InferenceResponse, ToolCall (shared types)
  compatible.go   # OpenAICompatibleClient — single impl for all /v1/chat/completions
  registry.go     # ProviderSpec, registry slice, LookupProvider
  models.go       # TopModels (~30), DefaultModelForProvider, ListModelsByProvider
  factory.go      # NewClientFromConfig — uses registry, provider-aware model defaults
  stub.go         # StubClient
  anthropic.go    # AnthropicClient — only when custom API needed
  chatjimmy.go    # ChatJimmyClient — chatjimmy.ai /api/chat; no auth
```

**Removed:** `openai.go` (merged into `compatible.go`). No per-provider files for OpenAI-compatible APIs.

### 6.1 Top 30 Models Support

- **Providers:** openai, conway, ollama, openrouter, groq, mistral, deepseek, xai, together, fireworks, perplexity, cohere, qwen, moonshot, chatjimmy (15 total).
- **TopModels:** ~30 globally popular models (GPT-4o, Claude, Gemini, Llama, Mistral, DeepSeek, Grok, Qwen, etc.) with default provider per model.
- **DefaultModelForProvider:** When `inferenceModel` is empty, factory uses provider-specific default (e.g. groq → llama-3.3-70b-versatile).

---

## 7. References

- mormclaw: `src/providers/traits.rs`, `compatible.rs`, `chatjimmy.rs`, `mod.rs`
- mormoneyOS: `internal/inference/`, `docs/design/ts-go-alignment.md`
