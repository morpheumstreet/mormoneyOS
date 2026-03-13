#!/bin/sh
# MoneyClaw / mormoneyOS Installer
# curl -fsSL https://raw.githubusercontent.com/morpheumlabs/mormoneyOS/main/scripts/moneyclaw.sh | sh
set -e

REPO="${MORMONEYOS_REPO:-https://github.com/morpheumlabs/mormoneyOS.git}"

if [ -n "$AUTOMATON_DIR" ]; then
  INSTALL_DIR="$AUTOMATON_DIR"
elif [ -w /opt ] || [ "$(id -u)" = "0" ]; then
  INSTALL_DIR="/opt/moneyclaw"
else
  INSTALL_DIR="$HOME/.automaton/runtime"
fi

if ! command -v go >/dev/null 2>&1; then
  echo "[ERROR] Go 1.21+ is required. Install it first: https://go.dev/doc/install" >&2
  exit 1
fi

if ! command -v git >/dev/null 2>&1; then
  echo "[ERROR] git is required." >&2
  exit 1
fi

if [ -d "$INSTALL_DIR/.git" ]; then
  echo "[INFO]  Updating existing installation at $INSTALL_DIR..."
  cd "$INSTALL_DIR" && git pull --ff-only
else
  echo "[INFO]  Cloning mormoneyOS to $INSTALL_DIR..."
  mkdir -p "$(dirname "$INSTALL_DIR")"
  git clone "$REPO" "$INSTALL_DIR"
  cd "$INSTALL_DIR"
fi

echo "[INFO]  Building moneyclaw..."
GOWORK=off go build -o bin/moneyclaw ./cmd/moneyclaw

echo "[INFO]  Launching moneyclaw run..."
exec ./bin/moneyclaw run
