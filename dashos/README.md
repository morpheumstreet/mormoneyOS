# DashOS

**MoneyClaw Command Center** — Bun-built React dashboard with wallet auth, webmormos design, and full access to the working task.

## Features

- **Working Task Dashboard**: Agent state, P&L, risk, LLM cost, active strategies, control (Pause/Resume/Refresh), Agent Comm Link (chat)
- **Configuration**: Limited config editing (when backend exposes `GET/PUT /api/config`)
- **Wallet Auth**: Connect browser wallet extension (MetaMask, Phantom, etc.), sign SIWE message → bearer token for write access
- **Design**: Electric system UI from webmormos (hero panels, electric cards, pairing shell, status pills)

## Setup

```bash
cd dashos
bun install
bun run dev
```

Dashboard runs at `http://localhost:5174`. It proxies `/api` to `http://localhost:8080` (moneyclaw).

### Auth

- **No `VITE_AUTH_API_URL`**: Connect wallet to enter. Write ops work if MoneyClaw has no auth (current default).
- **With `VITE_AUTH_API_URL`** (e.g. `https://api.conway.tech`): Full SIWE flow — sign message → receive bearer token → use for write requests.

Example:

```bash
VITE_AUTH_API_URL=https://api.conway.tech bun run dev
```

## Build

Uses `bun scripts/build.js` (no bunx) — follows the prologue build pattern. Tailwind v4 via `@tailwindcss/cli` (standalone, no Vite plugin).

```bash
bun run build          # Standalone (base /) — for preview or custom hosting
bun run build:embed    # For mormoneyOS embed (base /static/) — use `make web` from repo root
```

Output in `dist/`. For embedding in mormoneyOS, run `make web` from the repo root (builds with `/static/` base and copies to `internal/web/static`).

## API Proxy

Dev server proxies `/api` to `http://localhost:8080` (moneyclaw). Run `moneyclaw run` in another terminal.

### Dev bypass (agent browser / no wallet)

When `MONEYCLAW_DEV_BYPASS=1`, start moneyclaw and use the "Dev bypass (no wallet)" button on the connect screen, or trigger via JSON:

```bash
MONEYCLAW_DEV_BYPASS=1 moneyclaw run
```

Then from agent browser: `POST /api/auth/dev-bypass` → store `token` in `localStorage.setItem("dashos:bearer", token)` → reload.
