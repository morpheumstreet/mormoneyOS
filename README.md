# MoneyClaw (mormoneyOS)

**7x24 AI Agent that saves and makes money autonomously.**

Go implementation aligned with [moneyclaw-py](https://github.com/Qiyd81/moneyclaw-py) — sovereign AI agent runtime with web dashboard, skills, and survival economics.

Design reference: [mormoneyOS/docs/design](../mormoneyOS/docs/design)

## Structure

```
mormoneyOS-go/
├── cmd/
│   ├── moneyclaw/main.go    # Entry point
│   ├── root.go              # Cobra root + viper
│   ├── run.go               # run: bootstrap + main loop + web dashboard
│   ├── setup.go             # setup: wizard
│   ├── status.go            # status
│   ├── strategies.go        # strategies: list (placeholder)
│   ├── cost.go              # cost: LLM cost summary (placeholder)
│   ├── pause.go             # pause: via web API
│   ├── resume.go            # resume: via web API
│   └── init.go              # init: create ~/.automaton
├── internal/
│   ├── agent/               # ReAct loop, policy engine
│   ├── config/              # Config load/save
│   ├── conway/              # Conway API client, credits
│   ├── heartbeat/           # Daemon, scheduler, tasks
│   ├── state/               # SQLite schema, database
│   ├── types/               # Shared types
│   └── web/                 # Web dashboard (HTMX-style, moneyclaw-py aligned)
│       ├── server.go        # HTTP server, API routes
│       └── static/          # Embedded HTML/CSS/JS
├── go.mod
└── README.md
```

## Commands

| Command | Description |
|---------|-------------|
| `moneyclaw run` | Start runtime (agent loop + heartbeat + web dashboard) |
| `moneyclaw run --no-web` | Run without web dashboard |
| `moneyclaw run --no-telegram` | Run without Telegram (placeholder) |
| `moneyclaw setup` | Interactive setup wizard |
| `moneyclaw status` | Show config/DB status |
| `moneyclaw strategies` | List discovered strategies (placeholder) |
| `moneyclaw cost` | LLM cost summary (placeholder) |
| `moneyclaw pause` | Pause agent via web API |
| `moneyclaw resume` | Resume agent via web API |
| `moneyclaw init` | Create ~/.automaton |

## Web Dashboard

The web dashboard is available at `http://localhost:8080` by default when running `moneyclaw run`.

- **Status**: Agent state, P&L, risk level, tick count
- **Strategies**: Active strategies (placeholder list)
- **Control**: Pause / Resume
- **Chat**: Simple agent chat (status, help)
- **API**: REST endpoints aligned with moneyclaw-py (`/api/status`, `/api/strategies`, `/api/cost`, `/api/risk`, `/api/pause`, `/api/resume`, `/api/chat`)

## Config

- **Path:** `~/.automaton/automaton.json`
- **Env:** `AUTOMATON_DIR`, `CONWAY_API_URL`, `CONWAY_API_KEY`

## Build

```bash
go build -o moneyclaw ./cmd/moneyclaw
```

## Design Alignment

- Bootstrap sequence per [runtime-lifecycle.md](../mormoneyOS/docs/design/runtime-lifecycle.md)
- Policy engine with 6 rule categories per [security-model.md](../mormoneyOS/docs/design/security-model.md)
- Heartbeat daemon with durable scheduler per [modules.md](../mormoneyOS/docs/design/modules.md)
- Config, extension points per [extension-points.md](../mormoneyOS/docs/design/extension-points.md)
