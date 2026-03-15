#!/usr/bin/env bun

import { execSync } from "child_process";
import { existsSync, mkdirSync } from "fs";
import { join } from "path";

const DASHOS_DIR = process.cwd();
const TAILWIND_CLI = join(DASHOS_DIR, "node_modules", "@tailwindcss", "cli", "dist", "index.mjs");
const INPUT = join(DASHOS_DIR, "src", "index.css");
const OUTPUT = join(DASHOS_DIR, "public", "styles.css");
const watch = process.argv.includes("--watch");

try {
  if (!existsSync(INPUT)) {
    console.error("❌ Input CSS not found:", INPUT);
    process.exit(1);
  }
  const outDir = join(DASHOS_DIR, "public");
  if (!existsSync(outDir)) {
    mkdirSync(outDir, { recursive: true });
  }
  const watchArg = watch ? "--watch" : "";
  console.log("🔄 Building Tailwind CSS...");
  execSync(
    `bun "${TAILWIND_CLI}" -i "${INPUT}" -o "${OUTPUT}" ${watchArg}`.trim(),
    { stdio: "inherit", cwd: DASHOS_DIR }
  );
  if (!watch) {
    console.log("✅ Tailwind CSS built successfully!");
  }
} catch (error) {
  console.error("❌ Tailwind build failed:", error.message);
  process.exit(1);
}
