# Structured / Versioned Prompt Templates + Chain-of-Thought Forcing

**Date:** 2026-03-17  
**Purpose:** Replace ad-hoc prompt building with versioned, templated prompts and explicit Chain-of-Thought (CoT) forcing. Builds on [token-caps-truncation.md](./token-caps-truncation.md) and [context-trimming-stage2.md](./context-trimming-stage2.md).

---

## 1. Goals

| Goal | Description |
|------|-------------|
| **Versioned prompts** | Replace ad-hoc string concatenation in `agent/prompt.go` with versioned, templated prompts |
| **Explicit CoT** | Force structured reasoning: Thought → Risk → Plan → Action in every turn |
| **Token safety** | Keep total input under the cap enforced by `token.go` / `trim.go` |
| **Zero-downtime switching** | Allow prompt version switching via config (hot-reload friendly) |
| **Single source & testable** | Single source of truth, full test coverage, no duplication |

---

## 2. Package Structure

```
internal/
├── prompts/
│   ├── prompts.go          # Public API: Version, SystemPromptData, BuildSystemPrompt, GetCoTFooter
│   ├── loader.go           # go:embed + versioned template loading
│   ├── builder.go          # FormatHistoryForReAct, RenderReactCoT (no agent dependency)
│   ├── templates/
│   │   └── v1/
│   │       ├── system.tmpl      # Constitution, status, genesis, skills, CoT instructions
│   │       ├── react_cot.tmpl   # Memory + history + input + instructions (combined format)
│   │       └── output_format.tmpl  # JSON schema for tool calls / final answer
│   └── templates_test.go
└── agent/
    ├── prompts_integration.go   # BuildMessagesFromPrompts (orchestrates prompts + MessageTrimmer)
    └── prompts_integration_test.go
```

- **`//go:embed`** for zero-runtime file I/O
- Package is **internal** (private to the project)
- **No import cycle:** `prompts` does not import `agent`; integration lives in `agent`

---

## 3. Template Files

All templates use `text/template` syntax. Each is small (<400 tokens) and composable.

### 3.1 system.tmpl

Constitution, self-preservation rules, status block, genesis, skills, and CoT instructions.

**Template variables:**

| Variable | Source |
|----------|--------|
| `State` | Agent state (running, sleeping, low_compute, etc.) |
| `Credits` | Formatted credits string (e.g. `"12.50"`) |
| `Tier` | Survival tier (high, normal, low_compute, critical, dead) |
| `TurnCount` | Total turns |
| `Model` | Inference model ID |
| `LineageSummary` | Optional lineage from replication |
| `SkillsBlock` | Formatted enabled skills (from `skills.FormatForPrompt`) |
| `GenesisPrompt` | Truncated genesis purpose (max 2000 chars) |

**CoT section (embedded):**

```
## Response Format (Chain-of-Thought)
For every turn, structure your reasoning explicitly:
1. **Thought:** Think step-by-step about the situation.
2. **Risk:** Assess risks and policy compliance before acting.
3. **Plan:** Outline the action plan.
4. **Action:** Either call a tool OR output FinalAnswer in JSON.
```

### 3.2 react_cot.tmpl

Combined context format: memory block + conversation history + current input + instructions.

**Template variables:**

| Variable | Description |
|----------|-------------|
| `MemoryBlock` | Formatted memory from `memory.Retrieve` / `RetrieveWithBudget` |
| `History` | Formatted turns (e.g. via `FormatHistoryForReAct`) |
| `CurrentInput` | Pending user input |

Used for alternative single-message format. Primary flow uses multi-turn messages with CoT footer appended to last user message.

### 3.3 output_format.tmpl

JSON schema instructions for tool calls and final answer. Can be embedded in system prompt or used standalone.

---

## 4. Core Implementation

### 4.1 prompts.BuildSystemPrompt

```go
func BuildSystemPrompt(version Version, data SystemPromptData) (string, error)
```

Renders the system prompt from the template. Returns error for unsupported versions.

### 4.2 prompts.GetCoTFooter

```go
func GetCoTFooter() string
```

Returns the CoT instruction footer appended to the last user message. Kept small to avoid token bloat.

### 4.3 agent.BuildMessagesFromPrompts

```go
func BuildMessagesFromPrompts(
    ctx context.Context,
    version prompts.Version,
    systemData prompts.SystemPromptData,
    recentTurns []state.Turn,
    pendingInput string,
    memoryRetriever memory.MemoryRetriever,
    toolDefs []inference.ToolDefinition,
    limits TokenLimits,
    effectiveCap int,
    tok Tokenizer,
    log *slog.Logger,
) ([]inference.ChatMessage, error)
```

1. Renders system prompt via `prompts.BuildSystemPrompt`
2. Appends `prompts.GetCoTFooter` to `pendingInput`
3. Delegates to `MessageTrimmer.Trim` (or `BuildMessagesSafe` when no memory retriever)
4. Returns messages ready for inference

---

## 5. Integration with Agent Loop

### 5.1 Loop Logic

When `LoopConfig.PromptVersion != ""` (e.g. `"v1"`):

1. Build `SystemPromptData` from runtime state (credits, tier, lineage, skills, genesis)
2. Call `BuildMessagesFromPrompts` instead of `MessageTrimmer.Trim` / `BuildMessagesSafe`
3. Pass resulting messages to inference

When `PromptVersion` is empty: legacy path (ad-hoc `BuildSystemPrompt` + `MessageTrimmer`).

### 5.2 Zero Breaking Changes

- `token.go`, `trim.go`, `memory`, `inference` — unchanged
- `BuildMessagesSafe`, `MessageTrimmer` — reused
- Config: `promptVersion: "v1"` enables versioned prompts

---

## 6. Configuration

### 6.1 Config Fields

| Field | Location | Description |
|-------|----------|-------------|
| `promptVersion` | `automaton.json` | `"v1"` = versioned templates + CoT; empty = legacy |
| `PromptVersion` | `LoopConfig` | Passed from `AutomatonConfig` |

### 6.2 Example

```json
{
  "name": "automaton",
  "promptVersion": "v1",
  "inferenceModel": "llama3.1-8B"
}
```

---

## 7. Testing

| File | Tests |
|------|-------|
| `internal/prompts/templates_test.go` | `TestBuildSystemPrompt_V1`, `TestBuildSystemPrompt_UnsupportedVersion`, `TestGetCoTFooter`, `TestRenderReactCoT`, `TestFormatHistoryForReAct` |
| `internal/agent/prompts_integration_test.go` | `TestBuildMessagesFromPrompts_NoMemory`, `TestBuildMessagesFromPrompts_UnsupportedVersion` |

See [reports/index.md](../reports/index.md) for full traceability (PR1–PR6, A30–A31).

---

## 8. Rationale

| Aspect | Application |
|--------|-------------|
| Predictable | Versioned, templated, token-safe, CoT-forced → predictable reasoning |
| Single source | Single source of truth (templates), reuses token, trim, memory, context |
| Structure | Small package, clear public API, embed-based, follows Go layout |
| Future-proof | Adding v2 or self-critique = new template folder + one line in registry |

---

## 9. Related Documents

- [token-caps-truncation.md](./token-caps-truncation.md) — Token counting, `BuildMessagesSafe`
- [context-trimming-stage2.md](./context-trimming-stage2.md) — HistoryTrimmer, MessageTrimmer, TieredMemoryRetriever
- [memory-retrieval-step6.md](./memory-retrieval-step6.md) — Memory injection
- [reports/index.md](../reports/index.md) — Test report, prompts traceability
