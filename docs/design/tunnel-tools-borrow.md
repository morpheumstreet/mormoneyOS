# Tunnel Tools Design — Borrowing from mormclaw

**Date:** 2026-03-13  
**Purpose:** Design how to borrow mormclaw's tunnel feature into mormoneyOS as **built-in agent tools**, enabling cost-effective localhost exposure. **Optimized for extensibility:** service providers register their tunnel implementations and contribute tools via the tool registry system.

---

## 1. Executive Summary

mormclaw exposes a **Tunnel** trait and multiple providers (Cloudflare, Tailscale, ngrok, custom) to expose the gateway publicly. mormoneyOS has no gateway in that sense but runs a **web dashboard** on a configurable port. The agent may need to **expose localhost to the internet** for:

- Webhooks (Conway, external services)
- Remote access to the dashboard
- Child automaton callbacks
- MCP or other inbound integrations

**Design goals:**

1. **Provider-extensible:** Different tunnel providers (bore, cloudflare, ngrok, custom) register themselves; new providers can be added without modifying core code.
2. **Tool registry alignment:** Tunnel tools integrate via the same `Register()` / `RegisterMany()` / config / plugin mechanisms as Conway and other services.
3. **Service provider pattern:** Tunnel is a **service provider** that contributes tools to the registry when configured; other services (Conway, future Slack, etc.) follow the same pattern.
4. **Extensible:** Single registration path, split responsibilities, shared base for command-based providers.

**Cost-effective options** (vs paid ngrok):

| Provider   | Cost        | Auth        | Notes                                      |
|-----------|-------------|-------------|--------------------------------------------|
| **bore**  | Free        | None        | `bore local {port} --to bore.pub`          |
| **localtunnel** | Free | None   | `lt --port {port}`                         |
| **cloudflare** | Free tier | Token   | `cloudflared tunnel run --token ...`       |
| **tailscale**  | Free personal | Tailnet | `tailscale funnel {port}`              |
| **ngrok** | Free tier / paid | Auth token | Full-featured, rate limits on free |
| **custom**| Varies      | Depends     | Any command with `{port}` / `{host}`      |

---

## 2. mormclaw Tunnel Design (Reference)

### 2.1 Trait Contract

```rust
pub trait Tunnel: Send + Sync {
    fn name(&self) -> &str;
    async fn start(&self, local_host: &str, local_port: u16) -> Result<String>;
    async fn stop(&self) -> Result<()>;
    async fn health_check(&self) -> bool;
    fn public_url(&self) -> Option<String>;
}
```

### 2.2 Providers

| Provider   | Binary       | Config                          |
|-----------|--------------|----------------------------------|
| none      | —            | No tunnel                        |
| cloudflare| `cloudflared`| `[tunnel.cloudflare]` token      |
| tailscale | `tailscale`  | `[tunnel.tailscale]` funnel, hostname |
| ngrok     | `ngrok`      | `[tunnel.ngrok]` auth_token, domain |
| custom    | Any          | `[tunnel.custom]` start_command, health_url, url_pattern |

### 2.3 Custom Tunnel

- **Placeholders:** `{port}`, `{host}` in `start_command`
- **Examples:**
  - `bore local {port} --to bore.pub`
  - `lt --port {port}`
  - `ssh -R 80:localhost:{port} serveo.net`
- **URL extraction:** Optional `url_pattern` to parse public URL from stdout
- **Health:** Optional `health_url` to poll for liveness

### 2.4 Integration Point

In mormclaw, the tunnel is **runtime infrastructure**: the gateway starts the tunnel after binding its local port. It is **not** an agent tool — it is a config-driven service.

---

## 3. mormoneyOS Adaptation: Tools, Not Runtime

mormoneyOS does not have a gateway that needs a tunnel at startup. Instead, the **agent** may need to expose a port on demand. Therefore we model tunnels as **agent tools**:

| mormclaw              | mormoneyOS                          |
|-----------------------|--------------------------------------|
| Runtime tunnel (gateway) | Agent tools (`expose_port`, `remove_port`) |
| Config-driven provider | Tool args: `provider`, `port`, optional config |
| Single tunnel per run  | Multiple tunnels per port (or one active per port) |
| Trait + factory       | Tool interface + provider registry   |

### 3.1 Tool Mapping

| Stub (current)   | Real tool (proposed) | Description                                      |
|------------------|----------------------|--------------------------------------------------|
| `expose_port`    | `expose_port`        | Start a tunnel for a local port; return public URL |
| `remove_port`    | `remove_port`        | Stop the tunnel for a port                       |

We **replace** the stubs with real implementations.

---

## 4. Tool Design

### 4.1 `expose_port`

**Parameters:**

```json
{
  "type": "object",
  "properties": {
    "port": { "type": "integer", "description": "Local port to expose (e.g. 8080 for web dashboard)" },
    "provider": {
      "type": "string",
      "description": "Tunnel provider. Enum is dynamic from registered providers (bore, localtunnel, cloudflare, tailscale, ngrok, custom). Prefer bore or localtunnel for cost-effectiveness."
    },
    "host": {
      "type": "string",
      "description": "Local host (default: 127.0.0.1)"
    },
    "custom_command": {
      "type": "string",
      "description": "For provider=custom: command with {port} and {host} placeholders"
    }
  },
  "required": ["port"]
}
```

**Behavior:**

1. Resolve provider (default: `bore` if available, else `localtunnel`, else `custom` from config).
2. Build start command from provider template or `custom_command`.
3. Spawn subprocess, replace `{port}` and `{host}`.
4. Parse stdout/stderr for public URL (provider-specific or config `url_pattern`).
5. Store active tunnel in registry (port → process handle, public_url).
6. Return `{"public_url": "https://...", "provider": "bore", "port": 8080}`.

**Provider templates (built-in):**

| Provider     | Command template                                      |
|-------------|--------------------------------------------------------|
| bore        | `bore local {port} --to bore.pub`                      |
| localtunnel | `lt --port {port}` or `npx localtunnel --port {port}`  |
| cloudflare  | `cloudflared tunnel --no-autoupdate run --token $TOKEN --url http://localhost:{port}` |
| tailscale   | `tailscale funnel {port}`                              |
| ngrok       | `ngrok http {port}` (requires prior `ngrok config add-authtoken`) |
| custom      | From `custom_command` or config `tunnel.custom.start_command` |

### 4.2 `remove_port`

**Parameters:**

```json
{
  "type": "object",
  "properties": {
    "port": { "type": "integer", "description": "Port whose tunnel to stop" }
  },
  "required": ["port"]
}
```

**Behavior:**

1. Look up active tunnel for port.
2. Kill subprocess.
3. Remove from registry.
4. Return `{"removed": true, "port": 8080}`.

### 4.3 `tunnel_status` (optional)

**Parameters:** `{}`

**Behavior:** List all active tunnels: `[{port, provider, public_url}]`.

---

## 5. Config Extensions

To align with mormclaw and support custom/cloudflare/ngrok:

```json
{
  "tunnel": {
    "defaultProvider": "bore",
    "providers": {
      "bore": { "enabled": true },
      "localtunnel": { "enabled": true },
      "cloudflare": {
        "enabled": false,
        "token": "${CLOUDFLARE_TUNNEL_TOKEN}"
      },
      "tailscale": { "enabled": false },
      "ngrok": {
        "enabled": false,
        "authToken": "${NGROK_AUTH_TOKEN}"
      },
      "custom": {
        "enabled": false,
        "startCommand": "bore local {port} --to bore.pub",
        "healthUrl": null,
        "urlPattern": "https://"
      }
    }
  }
}
```

- **defaultProvider:** Used when agent omits `provider` in `expose_port`.
- **Secrets:** Prefer env vars; never log tokens.

---

## 6. Service Provider Pattern & Tool Registry Alignment

### 6.1 ServiceProvider Interface

To align with Conway and future services, introduce a **ServiceProvider** abstraction. A service provider contributes tools to the registry when it is configured:

```go
// ServiceProvider is implemented by services (Conway, Tunnel, future Slack, etc.)
// that contribute tools to the tool registry.
type ServiceProvider interface {
    Name() string
    Tools() []Tool
}
```

- **Conway** → `ConwayServiceProvider{Client}` implements `ServiceProvider`; `Tools()` returns `check_credits`, `list_sandboxes`, `list_models`, etc.
- **Tunnel** → `TunnelServiceProvider{Registry}` implements `ServiceProvider`; `Tools()` returns `expose_port`, `remove_port`, `tunnel_status`.

### 6.2 RegistryOptions Extension (Single Path)

**One registration path.** Conway, Tunnel, and future services all use `ServiceProviders`. No special-case `if opts.Conway != nil`.

```go
type RegistryOptions struct {
    Store           ToolStore
    Name            string
    ConfigTools     []types.ConfigToolDef
    InstalledDB     InstalledToolDB
    PluginPaths     []string
    ServiceProviders []ServiceProvider  // Conway, Tunnel, etc. — single path
}

// Conway and Tunnel are built as ServiceProviders and appended to opts.ServiceProviders
// before NewRegistryWithOptions. No Conway-specific branch in registry init.
```

Registration flow:

```go
// In NewRegistryWithOptions — single loop:
for _, sp := range opts.ServiceProviders {
    r.RegisterMany(sp.Tools())
}
```

### 6.3 Split Responsibilities

**ProviderRegistry** — register and list providers. **TunnelManager** — start, stop, status. Tools depend only on `TunnelManager`.

```go
// ProviderRegistry: register/list providers. No lifecycle.
type ProviderRegistry interface {
    Register(p TunnelProvider)
    Get(name string) (TunnelProvider, bool)
    List() []string
}

// TunnelManager: start/stop tunnels, query status. Uses ProviderRegistry.
type TunnelManager interface {
    Start(ctx context.Context, provider, host string, port int) (publicURL string, err error)
    Stop(port int) error
    Status() []ActiveTunnel
}

// TunnelProvider — minimal interface per provider.
type TunnelProvider interface {
    Name() string
    Start(ctx context.Context, host string, port int) (publicURL string, err error)
    Stop(port int) error
    IsAvailable() bool
}
```

**Interface Segregation:** Tools need only `TunnelManager`. Provider registration is a bootstrap concern.

### 6.4 CommandTunnelProvider Base

bore, localtunnel, cloudflare, ngrok, and custom all spawn a subprocess and parse stdout/stderr for the public URL. A shared base eliminates duplication:

```go
// CommandTunnelProvider — base for providers that run a CLI command.
// Subclasses supply: CommandTemplate, URLPattern (regex or substring), optional Env.
type CommandTunnelProvider struct {
    Name            string
    CommandTemplate string   // "bore local {port} --to bore.pub"
    URLPattern      string   // "https://" or regex to extract URL from stdout
    Binary          string   // "bore" — for IsAvailable() check
    Env             []string // optional, e.g. NGROK_AUTH_TOKEN
}

func (c *CommandTunnelProvider) Start(ctx context.Context, host string, port int) (string, error)
func (c *CommandTunnelProvider) Stop(port int) error
func (c *CommandTunnelProvider) IsAvailable() bool  // exec.LookPath(c.Binary)
```

bore, localtunnel, custom extend this; cloudflare/ngrok add token handling. Subprocess spawn + URL extraction live in one place.

### 6.5 Extensibility

A plugin or external package can:

1. Implement `TunnelProvider` for a new service (e.g. `frp`, `serveo`).
2. Call `providerRegistry.Register(&MyTunnelProvider{})` at init or via config.
3. No changes to `expose_port` / `remove_port` — they delegate to `TunnelManager`.

### 6.6 Config-Driven Provider Registration

```json
{
  "tunnel": {
    "defaultProvider": "bore",
    "providers": {
      "bore": { "enabled": true },
      "localtunnel": { "enabled": true },
      "cloudflare": { "enabled": false, "token": "${CLOUDFLARE_TUNNEL_TOKEN}" },
      "custom": {
        "enabled": true,
        "startCommand": "bore local {port} --to bore.pub"
      }
    }
  }
}
```

At startup: for each `enabled: true` provider, instantiate and register it. `ProviderRegistry.List()` reflects only enabled providers. `ExposePortTool.Parameters()` builds the `provider` enum dynamically from `List()` so the agent only sees available options.

### 6.7 Plugin / DB Extension

- **Plugins:** Export `RegisterTunnelProviders(register func(TunnelProvider))`. Loader forwards to `ProviderRegistry.Register()`.
- **installed_tools:** A tool of type `tunnel_provider` with config could register a custom provider at runtime (advanced).

### 6.8 Cross-Service Alignment

| Service   | ServiceProvider        | Contributed Tools                          |
|-----------|-------------------------|--------------------------------------------|
| Conway    | ConwayServiceProvider   | check_credits, list_sandboxes, list_models, heartbeat_ping |
| Tunnel    | TunnelServiceProvider   | expose_port, remove_port, tunnel_status    |
| (future)  | SlackServiceProvider    | slack_post, slack_list_channels            |
| (future)  | MCPServiceProvider      | mcp_invoke, mcp_list_tools                  |

All service providers use the same registration path: `opts.ServiceProviders` → `r.RegisterMany(sp.Tools())`. The tool registry remains flat and provider-agnostic.

---

## 7. Construction & Bootstrap

**Explicit construction.** Config → providers → service → registry. All in one place.

```
cmd/run.go (or config/bootstrap):
    1. cfg := config.Load()
    2. tunnelCfg := cfg.Tunnel  // from automaton.json
    3. providerRegistry, tunnelManager := tunnel.NewFromConfig(tunnelCfg)
    4. tunnelServiceProvider := tunnel.NewServiceProvider(tunnelManager, providerRegistry)
    5. opts.ServiceProviders = append(opts.ServiceProviders, tunnelServiceProvider)
    6. if opts.Conway != nil { opts.ServiceProviders = append(opts.ServiceProviders, conway.NewServiceProvider(opts.Conway)) }
    7. registry := tools.NewRegistryWithOptions(opts)
```

```go
// internal/tunnel/bootstrap.go
func NewFromConfig(cfg *TunnelConfig) (ProviderRegistry, TunnelManager) {
    reg := NewProviderRegistry()
    mgr := NewTunnelManager(reg)
    for name, p := range cfg.Providers {
        if !p.Enabled { continue }
        provider := buildProvider(name, p)  // bore, cloudflare, custom, etc.
        reg.Register(provider)
    }
    return reg, mgr
}
```

**Single responsibility:** `NewFromConfig` owns config → provider construction. Tools never see config.

---

## 8. Implementation Layout

### 8.1 Package Structure

```
internal/tunnel/
  provider.go     # TunnelProvider interface
  command.go      # CommandTunnelProvider (base for bore, localtunnel, custom)
  registry.go     # ProviderRegistry, ActiveTunnelStore
  manager.go      # TunnelManager (uses ProviderRegistry + ActiveTunnelStore)
  bootstrap.go    # NewFromConfig(cfg) → ProviderRegistry, TunnelManager
  service.go      # TunnelServiceProvider (implements ServiceProvider)
  tools.go        # ExposePortTool, RemovePortTool, TunnelStatusTool
  bore.go         # BoreProvider (extends CommandTunnelProvider)
  localtunnel.go  # LocaltunnelProvider
  cloudflare.go   # CloudflareProvider
  tailscale.go    # TailscaleProvider
  ngrok.go        # NgrokProvider
  custom.go       # CustomProvider
```

### 8.2 Tool Registration Flow

```
Config load
    → tunnel.NewFromConfig(cfg.Tunnel) → providerRegistry, tunnelManager
    → For each enabled provider: reg.Register(provider)
    → TunnelServiceProvider{Manager: tunnelManager, Registry: providerRegistry}
    → opts.ServiceProviders = append(..., tunnelServiceProvider, conwayServiceProvider, ...)
    → NewRegistryWithOptions(opts)
    → for _, sp := range opts.ServiceProviders { r.RegisterMany(sp.Tools()) }
    → expose_port, remove_port, tunnel_status registered
```

### 8.3 Policy

- **Risk level:** `caution` — spawning subprocesses and exposing ports.
- **Policy rule:** Same as shell — validate provider is in allowlist when configured.

---

## 9. Cost-Effectiveness Strategy

1. **Default to bore** — no auth, free, single binary.
2. **Fallback to localtunnel** — `npx localtunnel` or `lt` if bore unavailable.
3. **Cloudflare / Tailscale** — when user configures (free tiers).
4. **ngrok** — when user explicitly needs it (free tier limits).
5. **Custom** — user supplies command (e.g. self-hosted bore server, frp, serveo).

Agent prompt can suggest: *"Use expose_port with provider bore or localtunnel for cost-effective public URLs."*

---

## 10. Security Considerations

- **Subprocess execution:** Same policy as `shell` — validate provider is in allowlist.
- **Port range:** Optionally restrict to e.g. 8080–8099 (dashboard, webhooks).
- **Secrets:** Cloudflare token, ngrok token from env or config; never in tool args or logs.
- **URL exposure:** Public URL is returned to the agent; ensure it is not logged in sensitive contexts.

---

## 11. Phased Implementation

### Phase 1 — Minimal (bore + custom)

- [ ] `TunnelProvider` interface; `ProviderRegistry` + `TunnelManager` (split).
- [ ] `CommandTunnelProvider` base (subprocess spawn, `{port}`/`{host}` replace, URL extraction).
- [ ] `bore` provider (extends CommandTunnelProvider).
- [ ] `custom` provider (extends CommandTunnelProvider, command from config).
- [ ] `NewFromConfig`, `TunnelServiceProvider`, `ExposePortTool`, `RemovePortTool`.
- [ ] Config: `tunnel.defaultProvider`, `tunnel.providers.custom`.
- [ ] Conway refactored to `ServiceProvider`; single `ServiceProviders` loop in registry.

### Phase 2 — More Providers

- [ ] `localtunnel` provider (CommandTunnelProvider).
- [ ] `cloudflare` provider (token from config/env).
- [ ] `tailscale` provider.
- [ ] `ngrok` provider.
- [ ] `TunnelStatusTool`.

### Phase 3 — Robustness

- [ ] Health check (optional `health_url` in CommandTunnelProvider).
- [ ] Timeout and retry.
- [ ] Persist active tunnels to DB (optional; or keep in-memory only).

---

## 12. Summary

| Aspect | Application |
|--------|-------------|
| Single path | Single `ServiceProvider` registration path; `CommandTunnelProvider` base for bore/localtunnel/custom; shared subprocess + URL extraction logic. |
| Boundaries | Tools depend on `TunnelManager` interface; config → construction in `NewFromConfig`; secrets from env/config. |
| Responsibilities | `ProviderRegistry` = register/list; `TunnelManager` = start/stop/status; `CommandTunnelProvider` = subprocess + URL parse. |
| Extensibility | New providers via `Register()`; new services via `ServiceProviders`; no core edits. |
| Interfaces | `TunnelProvider` minimal; `TunnelManager` separate from `ProviderRegistry`; tools need only `TunnelManager`. |
| Dependencies | Tools → `TunnelManager`; Manager → `ProviderRegistry`; all via interfaces. |

---

## 13. References

- mormclaw: `src/tunnel/` (mod.rs, custom.rs, cloudflare.rs, tailscale.rs, ngrok.rs)
- mormclaw config: `TunnelConfig`, `CustomTunnelConfig` in schema
- mormoneyOS tools: `internal/tools/tools.go`, `stubs.go`
- bore: https://github.com/ekzhang/bore
- localtunnel: https://github.com/localtunnel/localtunnel
