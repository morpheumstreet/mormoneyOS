# @mormoneyOS/dashui

MoneyClaw Dashboard UI — React SPA aligned with [moneyclaw-py](https://github.com/Qiyd81/moneyclaw-py) design.

## Build

Uses **bun** for build only:

```bash
bun install
bun run build
```

Output: `dist/` (Vite build)

## Dev

```bash
bun run dev
```

Runs Vite dev server. Point API base to mormoneyOS backend (e.g. `http://localhost:8080`) via proxy or env.

## Structure

- `src/api.ts` — API client (status, strategies, cost, pause, resume, chat)
- `src/App.tsx` — Root layout
- `src/components/` — Header, StatsGrid, StrategiesList, ChatPanel, ControlPanel, StrategyModal
