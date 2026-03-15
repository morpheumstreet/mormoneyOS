#!/usr/bin/env bun

import { spawn, execSync } from "child_process";
import { join } from "path";

const DASHOS_DIR = process.cwd();

// Initial Tailwind build (Vite needs styles.css before serving)
execSync("bun scripts/build-tailwind.js", { stdio: "inherit", cwd: DASHOS_DIR });

// Spawn Tailwind watch
const tailwind = spawn("bun", ["scripts/build-tailwind.js", "--watch"], {
  cwd: DASHOS_DIR,
  stdio: "inherit",
});

// Spawn Vite dev server
const vite = spawn("bun", [join(DASHOS_DIR, "node_modules", "vite", "bin", "vite.js")], {
  cwd: DASHOS_DIR,
  stdio: "inherit",
});

function cleanup() {
  tailwind.kill();
  vite.kill();
  process.exit(0);
}

process.on("SIGINT", cleanup);
process.on("SIGTERM", cleanup);
