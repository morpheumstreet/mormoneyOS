# MormoneyOS (MoneyClaw)

**7x24 AI Agent that saves and makes money autonomously.**

Sovereign AI agent runtime with web dashboard, skills, and survival economics. Go implementation aligned with [moneyclaw-py](https://github.com/Qiyd81/moneyclaw-py).

---

## Install (one line)

**Requires:** [Docker](https://docs.docker.com/get-docker/)

```bash
curl -fsSL https://raw.githubusercontent.com/morpheumstreet/mormoneyOS/main/scripts/install-docker.sh | bash
```

This pulls the image, mounts `~/.automaton` for data, and starts the agent. Web dashboard at **http://localhost:8080**.

### Options

| Env var | Default | Description |
|---------|---------|-------------|
| `MORMONEYOS_BOT` | `default` | Bot name (each bot gets its own data dir) |
| `MORMONEYOS_PORT` | `8080` | Host port for web UI |
| `MORMONEYOS_DAEMON` | `0` | Set to `1` to run in background |
| `AUTOMATON_DIR` | `~/.automaton` or `~/.automaton-{BOT}` | Data directory |

**Multi-bot example** (run in separate terminals):

```bash
MORMONEYOS_BOT=trading  MORMONEYOS_PORT=8080  curl -fsSL https://raw.githubusercontent.com/morpheumstreet/mormoneyOS/main/scripts/install-docker.sh | bash
MORMONEYOS_BOT=research MORMONEYOS_PORT=8081  curl -fsSL https://raw.githubusercontent.com/morpheumstreet/mormoneyOS/main/scripts/install-docker.sh | bash
```

---

## Commands

| Command | Description |
|---------|-------------|
| `moneyclaw run` | Start runtime (agent loop + heartbeat + web dashboard) |
| `moneyclaw run --no-web` | Run without web dashboard |
| `moneyclaw setup` | Interactive setup wizard |
| `moneyclaw status` | Show config/DB status |
| `moneyclaw pause` | Pause agent via web API |
| `moneyclaw resume` | Resume agent via web API |
| `moneyclaw init` | Create ~/.automaton |

## Web Dashboard

At `http://localhost:8080`: Status, P&L, risk level, strategies, pause/resume, chat. REST API: `/api/status`, `/api/strategies`, `/api/cost`, `/api/risk`, `/api/pause`, `/api/resume`, `/api/chat`.

## Config

- **Path:** `~/.automaton/automaton.json`
- **Env:** `AUTOMATON_DIR`, `CONWAY_API_URL`, `CONWAY_API_KEY`

## Build from source

```bash
go build -o moneyclaw ./cmd/moneyclaw
```

---

## Appendix A: Design Introduction

The mormoneyOS design docs describe the architecture, policies, and subsystems of the sovereign AI agent runtime. They are intended for contributors and integrators who need to understand or extend the system.

### Core lifecycle & modules

| Doc | Description |
|-----|-------------|
| [ARCHITECTURE.md](ARCHITECTURE.md) | Full system overview, runtime lifecycle, security model, heartbeat daemon |
| [feature-lis.md](docs/design/feature-lis.md) | Go system feature table, agent loop detail |

### Agent & context

| Doc | Description |
|-----|-------------|
| [context-trimming-stage2.md](docs/design/context-trimming-stage2.md) | Message trimming, token budgets |
| [token-caps-truncation.md](docs/design/token-caps-truncation.md) | Token caps, truncation strategy |
| [prompt-templates-cot.md](docs/design/prompt-templates-cot.md) | Versioned prompts, chain-of-thought |
| [model-routing-reflexion.md](docs/design/model-routing-reflexion.md) | Model routing, reflection engine |
| [tool-system.md](docs/design/tool-system.md) | Tool registry, policy gating |

### Memory, skills & identity

| Doc | Description |
|-----|-------------|
| [memory-system-5-tier.md](docs/design/memory-system-5-tier.md) | 5-tier memory schema |
| [memory-auto-ingestion.md](docs/design/memory-auto-ingestion.md) | Auto-ingestion, extraction |
| [skills-design.md](docs/design/skills-design.md) | Skills architecture, packaging |
| [wallet-identity-architecture.md](docs/design/wallet-identity-architecture.md) | Wallet, identity, multi-chain |
| [mnemonic-wallet-multichain.md](docs/design/mnemonic-wallet-multichain.md) | BIP-39 mnemonic, key derivation |

### Channels, children & extensions

| Doc | Description |
|-----|-------------|
| [social-channel-design.md](docs/design/social-channel-design.md) | Conway, Telegram, Discord channels |
| [child-runtime-protocol.md](docs/design/child-runtime-protocol.md) | Child agents, spawn protocol |
| [tunnel-tools-borrow.md](docs/design/tunnel-tools-borrow.md) | Tunnel tools, localhost exposure |
| [mormclaw-provider-borrow.md](docs/design/mormclaw-provider-borrow.md) | AI provider architecture |
