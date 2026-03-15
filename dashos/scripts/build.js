#!/usr/bin/env bun

import { execSync } from "child_process";
import { join } from "path";

const DASHOS_DIR = process.cwd();
const VITE_BIN = join(DASHOS_DIR, "node_modules", "vite", "bin", "vite.js");

const embed = process.argv.includes("--embed");
const baseArg = embed ? "--base /static/" : "";

try {
  execSync("bun scripts/build-tailwind.js", { stdio: "inherit", cwd: DASHOS_DIR });

  console.log("🔄 Building dashos...");
  execSync(`bun "${VITE_BIN}" build ${baseArg}`.trim(), {
    stdio: "inherit",
    cwd: DASHOS_DIR,
  });
  console.log("✅ Build completed successfully!");
} catch (error) {
  console.error("❌ Build failed:", error.message);
  process.exit(1);
}
