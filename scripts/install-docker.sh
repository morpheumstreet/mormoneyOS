#!/usr/bin/env bash
# mormoneyOS (MoneyClaw) — one-line Docker install (supports multi-bot)
# Usage: curl -fsSL https://raw.githubusercontent.com/morpheumstreet/mormoneyOS/main/scripts/install-docker.sh | bash
#
# Single bot (default):
#   curl -fsSL ... | bash
#
# Multiple bots (each in its own container):
#   MORMONEYOS_BOT=trading  MORMONEYOS_PORT=8080  curl -fsSL ... | bash   # terminal 1
#   MORMONEYOS_BOT=research MORMONEYOS_PORT=8081  curl -fsSL ... | bash   # terminal 2
#   MORMONEYOS_BOT=social   MORMONEYOS_PORT=8082  curl -fsSL ... | bash   # terminal 3
#
# Env overrides:
#   MORMONEYOS_BOT    - Bot name (default: default). Each bot gets its own data dir and container.
#   MORMONEYOS_IMAGE  - Docker image (default: ghcr.io/morpheumlabs/mormoneyos)
#   MORMONEYOS_TAG    - Image tag (default: latest)
#   AUTOMATON_DIR     - Host path for data (default: $HOME/.automaton or $HOME/.automaton-{BOT})
#   MORMONEYOS_PORT   - Host port for web UI (default: 8080; use different ports per bot)
#   MORMONEYOS_PULL   - Set to 0 to skip pull (use local image)
#   MORMONEYOS_DAEMON - Set to 1 to run in background (-d)

set -e

BOT="${MORMONEYOS_BOT:-default}"
IMAGE="${MORMONEYOS_IMAGE:-ghcr.io/morpheumlabs/mormoneyos}"
TAG="${MORMONEYOS_TAG:-latest}"
PORT="${MORMONEYOS_PORT:-8080}"
DAEMON="${MORMONEYOS_DAEMON:-0}"
FULL_IMAGE="${IMAGE}:${TAG}"

# Data dir: explicit AUTOMATON_DIR, or ~/.automaton-{bot} for named bots, ~/.automaton for default
if [ -n "${AUTOMATON_DIR}" ]; then
  DATA_DIR="$AUTOMATON_DIR"
elif [ "$BOT" = "default" ]; then
  DATA_DIR="$HOME/.automaton"
else
  DATA_DIR="$HOME/.automaton-${BOT}"
fi

CONTAINER_NAME="mormoneyos-${BOT}"

if ! command -v docker >/dev/null 2>&1; then
  echo "[ERROR] Docker is required. Install from https://docs.docker.com/get-docker/" >&2
  exit 1
fi

echo "[INFO] mormoneyOS Docker — bot=${BOT} image=${FULL_IMAGE}"
echo "[INFO] Data: ${DATA_DIR}  Port: ${PORT}  Container: ${CONTAINER_NAME}"

mkdir -p "$DATA_DIR"

if [ "${MORMONEYOS_PULL:-1}" != "0" ]; then
  echo "[INFO] Pulling ${FULL_IMAGE}..."
  docker pull "$FULL_IMAGE"
fi

RUN_OPTS=(
  --rm
  --name "$CONTAINER_NAME"
  -p "${PORT}:8080"
  -v "${DATA_DIR}:/data"
  -e AUTOMATON_DIR=/data
  --user "$(id -u):$(id -g)"
)

if [ "$DAEMON" = "1" ]; then
  echo "[INFO] Starting mormoneyOS (bot=${BOT}) in background — http://localhost:${PORT}"
  docker run -d "${RUN_OPTS[@]}" "$FULL_IMAGE"
  echo "[INFO] Run 'docker logs -f ${CONTAINER_NAME}' to follow logs"
else
  echo "[INFO] Starting mormoneyOS (bot=${BOT}) — http://localhost:${PORT}"
  exec docker run -it "${RUN_OPTS[@]}" "$FULL_IMAGE"
fi
