# Tool System Design

**Date:** 2026-03-13  
**Purpose:** Flat, extensible tool design for the Go agent runtime.

---

## 1. Architecture

- **Flat design:** Each tool is a first-class entity with no hierarchy. Tools are identified by name only.
- **Self-contained:** Each tool owns its name, description, JSON schema (parameters), and execution logic.
- **Extensible:** New tools can be added via `Register()` or `RegisterMany()` without modifying core code.
- **Policy-uniform:** All tools (built-in, custom, future plugins) pass through the same policy engine.

---

## 2. Tool Contract

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() string  // JSON schema for parameters (OpenAI function format)
    Execute(ctx context.Context, args map[string]any) (string, error)
}
```

Each tool is responsible for:

- **Name:** Unique identifier (e.g. `shell`, `file_read`).
- **Description:** Natural-language description for the inference model.
- **Parameters:** JSON schema for the tool's arguments (OpenAI-compatible).
- **Execute:** Implementation that runs the tool and returns a string result.

---

## 3. Registry

The registry is an append-only map of name → tool:

```go
type Registry struct {
    tools map[string]Tool
}

func (r *Registry) Register(t Tool)           // Add one tool
func (r *Registry) RegisterMany(tools []Tool) // Add many (plugins, config-driven)
func (r *Registry) Alias(name, target string) // Map alias to existing tool
func (r *Registry) Execute(ctx, name, args) (string, error)
func (r *Registry) Schemas() []inference.ToolDefinition  // For inference
func (r *Registry) List() []string
```

- **Register:** Adds a tool by its canonical name.
- **RegisterMany:** Batch registration for plugins or config-driven expansion.
- **Alias:** Allows multiple names (e.g. `exec` → `shell`) to point to the same implementation.
- **Schemas:** Returns OpenAI-format tool definitions for inference; each registered name (including aliases) gets a schema.

---

## 4. Tool Categories

### 4.1 Base (always registered)
- `shell` / `exec`, `file_read`, `write_file`
- `git_status`, `git_diff`, `git_log`, `git_commit`, `git_push`, `git_branch`, `git_clone`

### 4.2 Store-dependent (when `RegistryOptions.Store` set)
- `sleep`, `system_synopsis`, `list_skills`

### 4.3 ServiceProvider-dependent (when `RegistryOptions.ServiceProviders` set)

**Single registration path.** Conway, Tunnel, and future services all use `ServiceProviders`.

- **Conway:** `check_credits`, `list_sandboxes`, `list_models`, `heartbeat_ping`
- **Tunnel:** `expose_port`, `remove_port`, `tunnel_status`
- **Future:** Slack, MCP, etc. contribute tools via the same pattern

### 4.4 Stubs (~52 tools)
Tools not yet implemented return "Not implemented in Go runtime yet." Schema and description match TS for future parity.

---

## 5. Extension Points

| Mechanism | Status | Description |
|-----------|--------|-------------|
| **Code** | ✅ | `Register()`, `RegisterMany()` |
| **ServiceProviders** | ✅ | Single path: `ServiceProvider` interface. Conway, Tunnel, future Slack/MCP implement it; registry iterates `opts.ServiceProviders` only. |
| **Config** | ✅ | Load tools from YAML/JSON via `tools` array or `toolsConfigPath` in automaton.json |
| **DB** | ✅ | `installed_tools` table (TS-aligned); `GetInstalledTools()`, `InstallTool()`, `RemoveTool()` |
| **Plugins** | ✅ | Load `.so` modules from `pluginPaths` (Linux only; `.wasm` future) |

### 5.0 ServiceProvider Pattern (Extensible)

**One registration path.** No Conway-specific or Tunnel-specific branches in registry init.

```go
type ServiceProvider interface {
    Name() string
    Tools() []Tool
}

// RegistryOptions.ServiceProviders = []ServiceProvider
// At init: for _, sp := range opts.ServiceProviders { r.RegisterMany(sp.Tools()) }
```

- **Conway:** `ConwayServiceProvider{Client}` implements `ServiceProvider`; `Tools()` returns Conway tools.
- **Tunnel:** `TunnelServiceProvider{Manager, Registry}` implements `ServiceProvider`; yields `expose_port`, `remove_port`, `tunnel_status`. Tunnel backends implement `TunnelProvider`; shared `CommandTunnelProvider` base. See [tunnel-tools-borrow.md](./tunnel-tools-borrow.md).

### 5.1 Config Tools

Add to `automaton.json`:

```json
{
  "tools": [
    {
      "name": "echo",
      "description": "Echo the given text.",
      "parameters": "{\"type\":\"object\",\"properties\":{\"text\":{\"type\":\"string\"}},\"required\":[\"text\"]}",
      "type": "shell",
      "command": "echo"
    }
  ],
  "toolsConfigPath": "~/.automaton/tools.yaml"
}
```

Or use a separate file (`tools.yaml` / `tools.json`):

```yaml
tools:
  - name: echo
    description: Echo the given text.
    parameters:
      type: object
      properties:
        text: { type: string }
      required: [text]
    type: shell
    command: echo
```

### 5.2 DB Installed Tools

Tools in `installed_tools` table are registered at startup. Use `InstallTool()` / `RemoveTool()` to manage. TS-aligned schema: `id`, `name`, `type`, `config`, `installed_at`, `enabled`.

### 5.3 Plugins

Set `pluginPaths` in config to directories containing `.so` files. Each plugin must export:

```go
func RegisterTools(register func(Tool))
```

Go plugins require Linux and matching Go version. `.wasm` support is planned.

---

## 6. Adding a New Tool

1. Implement the `Tool` interface.
2. Call `registry.Register(&MyTool{})` (or `RegisterMany`) before the agent loop starts.
3. The registry automatically includes the new tool in `Schemas()` and `Execute()`.

Example:

```go
type EchoTool struct{}
func (EchoTool) Name() string        { return "echo" }
func (EchoTool) Description() string { return "Echo the given text." }
func (EchoTool) Parameters() string  { return `{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}` }
func (EchoTool) Execute(ctx context.Context, args map[string]any) (string, error) {
    t, _ := args["text"].(string)
    return t, nil
}

registry.Register(&EchoTool{})
```

---

## 7. References

- [ts-go-alignment.md](./ts-go-alignment.md) — TS vs Go alignment; tool count gap
- [tunnel-tools-borrow.md](./tunnel-tools-borrow.md) — Tunnel tools design (borrowed from mormclaw; cost-effective expose_port/remove_port)
- [ARCHITECTURE.md](../../ARCHITECTURE.md) — Tool System section
