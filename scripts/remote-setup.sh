#!/usr/bin/env bash
set -euo pipefail

# Base path containing moneyman1, moneyman2, ...
BASE_DIR="${BASE_DIR:-/home/eflash31/clawlaundry}"

for dir in "${BASE_DIR}"/moneyman*; do
  [ -d "$dir" ] || continue
  name="$(basename "$dir")"
  echo "=== Setting up $name ($dir) ==="
  docker run --rm -it \
    -v "${dir}:/data" \
    -e AUTOMATON_DIR=/data \
    sorajez/mormoneyos:latest \
    setup
  chown -R 1000:1000 "$dir"
done

