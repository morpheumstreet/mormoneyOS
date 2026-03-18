# MCP (Model Context Protocol) Design

**Date:** 2026-03-18  
**Purpose:** Agent-native, standardized tool-calling surface at `/mcp` on port 8080.

---

## 1. Overview

MCP is mounted at `/mcp` on the same port 8080 as DashOS. Zero new servers, zero port mapping. Human-friendly REST marketplace endpoints will live at `/api/marketplace` (per MormAegis.pdf). Both share the single Docker container runtime.

**Conway stays separate.** Conway is core infrastructure (sandbox, model routing, credits, social relay, x402). It remains a `ServiceProvider` that registers its own tools. Conway capabilities are exposed *as optional MCP tools* (e.g. `mcp.conway_broadcast`) when needed — thin wrappers, not a merge.

---

## 2. Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/mcp/tools` | List available tools (MCP format) |
| POST | `/mcp` | Execute a tool by name |

**Request (POST /mcp):**
```json
{
  "name": "tool_name",
  "arguments": { "key": "value" }
}
```

**Response (POST /mcp):**
```json
{
  "content": [{ "type": "text", "text": "result or error message" }]
}
```

---

## 3. Architecture (Clean + DRY + SOLID)

```
internal/
├── mcp/                  # MCP adapter layer (thin, framework-specific)
│   ├── handler.go        # POST /mcp (execute) + GET /mcp/tools
│   ├── protocol/         # MCP spec types + tool schema
│   ├── dto/              # Request/Response models (DRY with REST)
│   └── tools/            # One file per tool (SRP)
│       ├── registry.go   # Bridge to existing Tool Registry
│       └── marketplace/  # search.go, install.go, etc. (Phase 1+)
├── marketplace/          # Core domain (Phase 1+)
│   ├── entity/
│   ├── usecase/
│   ├── port/
│   └── service/
└── web/                  # DashOS — /mcp routes registered here
```

- **Bridge:** MCP handler uses `ToolsLister` + `Executor` from the existing Tool Registry. No duplication.
- **Agent Card:** Future Phase 2 adds Agent Card signing middleware for `/mcp` (reuse existing `buildAgentCard`).
- **Policy:** Tool execution flows through the same policy engine as internal agent tools (when Executor is the Registry).

---

## 4. Phased Rollout

| Phase | Scope | Status |
|-------|-------|--------|
| **0** | Folders, routes, DTOs, MCP schema, GET/POST wired to Registry | ✅ Done |
| **0.5** | Marketplace skeleton, 7 mormaegis.* tools (discovery + stubs via ServiceProvider) | ✅ Done |
| **1** | Full tool implementations calling marketplace usecases | Pending |
| **2** | Full protocol, Agent Card signing, on-chain VC badge, Morpheum L1 | Pending |
| **3** | Conway bridge (thin wrappers), ARCHITECTURE.md update | Pending |
| **4** | MCP + REST share use cases, DashOS marketplace tab | Optional |

---

## 5. Dependencies

- **ToolsLister** (ServerConfig): Lists tools and schemas.
- **Executor** (ServerConfig): Executes tools by name. When set to the Registry, all 57+ tools are callable via MCP.

---

## 6. References

- [tool-system.md](./tool-system.md) — Tool Registry, ServiceProvider pattern
- [agent-cards.md](./agent-cards.md) — Agent Card, ERC-8004
- MormAegis.pdf — Marketplace, MCP tools, x402
