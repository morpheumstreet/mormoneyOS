#!/bin/sh
# MoneyClaw (mormoneyOS) — build and run from source
# Usage: ./scripts/automaton.sh   (from repo root)
# Or:    curl -fsSL https://raw.githubusercontent.com/morpheumlabs/mormoneyOS/main/scripts/automaton.sh | sh
set -e

REPO="${MORMONEYOS_REPO:-https://github.com/morpheumlabs/mormoneyOS.git}"

# Determine run directory
if [ -f "go.mod" ] && [ -d "cmd/moneyclaw" ]; then
  RUN_DIR="$(pwd)"
  FROM_REPO=true
else
  INSTALL_DIR="${MORMONEYOS_DIR:-$HOME/.mormoneyos/runtime}"
  if [ -d "$INSTALL_DIR/.git" ]; then
    echo "[INFO] Updating existing installation at $INSTALL_DIR..."
    cd "$INSTALL_DIR" && git pull --ff-only
  else
    echo "[INFO] Cloning mormoneyOS to $INSTALL_DIR..."
    mkdir -p "$(dirname "$INSTALL_DIR")"
    git clone "$REPO" "$INSTALL_DIR"
    cd "$INSTALL_DIR"
  fi
  RUN_DIR="$(pwd)"
  FROM_REPO=false
fi

# Preflight: Go
if ! command -v go >/dev/null 2>&1; then
  echo "[ERROR] Go 1.21+ is required. Install it from https://go.dev/dl/" >&2
  exit 1
fi

# Build
echo "[INFO] Building moneyclaw..."
cd "$RUN_DIR"
GOWORK=off go build -o bin/moneyclaw ./cmd/moneyclaw

# Run
echo "[INFO] Starting moneyclaw..."
exec ./bin/moneyclaw run
