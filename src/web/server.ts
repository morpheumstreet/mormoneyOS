/**
 * Web Dashboard Server
 *
 * HTTP server for MoneyClaw web dashboard, aligned with moneyclaw-py design.
 * Serves API endpoints and static assets at http://localhost:8080 by default.
 */

import http from "http";
import path from "path";
import { fileURLToPath } from "url";
import fs from "fs";
import type { AutomatonDatabase } from "../types.js";
import type { ConwayClient } from "../types.js";
import { createLogger } from "../observability/logger.js";
import { insertWakeEvent } from "../state/database.js";

const logger = createLogger("web");
const __dirname = path.dirname(fileURLToPath(import.meta.url));

// Static assets: packages/dashui/dist (monorepo) or dist/web/static (copied during build)
function resolveStaticDir(): string {
  const dashuiDist = path.resolve(__dirname, "../../../packages/dashui/dist");
  const localStatic = path.join(__dirname, "static");
  if (fs.existsSync(path.join(dashuiDist, "index.html"))) {
    return dashuiDist;
  }
  if (fs.existsSync(path.join(localStatic, "index.html"))) {
    return localStatic;
  }
  return localStatic;
}

const MIME_TYPES: Record<string, string> = {
  ".html": "text/html",
  ".css": "text/css",
  ".js": "application/javascript",
  ".json": "application/json",
  ".ico": "image/x-icon",
  ".svg": "image/svg+xml",
};

export interface WebServerContext {
  db: AutomatonDatabase;
  conway: ConwayClient;
  config: { name: string; version: string };
}

export function createWebServer(ctx: WebServerContext): http.Server {
  const { db, conway, config } = ctx;

  const server = http.createServer(async (req, res) => {
    const url = new URL(req.url || "/", `http://${req.headers.host}`);
    const pathname = url.pathname;

    // CORS headers for API
    res.setHeader("Access-Control-Allow-Origin", "*");
    res.setHeader("Access-Control-Allow-Methods", "GET, POST, OPTIONS");
    res.setHeader("Access-Control-Allow-Headers", "Content-Type");

    if (req.method === "OPTIONS") {
      res.writeHead(204);
      res.end();
      return;
    }

    // API routes
    if (pathname.startsWith("/api/")) {
      await handleApi(req, res, pathname, url, ctx);
      return;
    }

    const staticDir = resolveStaticDir();

    // SPA: serve index.html for / and /index.html
    if (pathname === "/" || pathname === "/index.html") {
      serveFile(res, path.join(staticDir, "index.html"));
      return;
    }

    // Static assets (Vite outputs to /assets/*)
    const filePath = path.join(staticDir, pathname.slice(1));
    if (fs.existsSync(filePath) && fs.statSync(filePath).isFile()) {
      serveFile(res, filePath);
      return;
    }

    // SPA fallback: client-side routing
    serveFile(res, path.join(staticDir, "index.html"));
  });

  return server;
}

async function handleApi(
  req: http.IncomingMessage,
  res: http.ServerResponse,
  pathname: string,
  url: URL,
  ctx: WebServerContext,
): Promise<void> {
  const { db, conway, config } = ctx;

  const sendJson = (data: unknown, status = 200) => {
    res.writeHead(status, { "Content-Type": "application/json" });
    res.end(JSON.stringify(data));
  };

  try {
    // GET /api/status
    if (pathname === "/api/status" && req.method === "GET") {
      const state = db.getAgentState();
      const turnCount = db.getTurnCount();
      const address = db.getIdentity("address") || "0x0";
      let creditsCents = 0;
      try {
        creditsCents = await conway.getCreditsBalance();
      } catch {
        creditsCents = parseInt(db.getKV("credits_cents") || "0", 10) || 0;
      }
      const walletValue = creditsCents / 100;

      const status = {
        is_running: state === "running" || state === "waking",
        state,
        tick_count: turnCount,
        wallet_value: walletValue,
        today_pnl: 0,
        dry_run: true,
        address,
        name: config.name,
        version: config.version,
      };
      sendJson(status);
      return;
    }

    // GET /api/strategies — map skills + children to strategy-like list
    if (pathname === "/api/strategies" && req.method === "GET") {
      const skills = db.getSkills(true);
      const children = db.getChildren().filter((c) => c.status !== "dead");

      const strategies = [
        ...skills.map((s) => ({
          name: s.name,
          description: s.description || "Skill",
          enabled: s.enabled,
          risk_level: "low" as const,
        })),
        ...children.map((c) => ({
          name: c.name,
          description: `Child automaton (${c.status})`,
          enabled: c.status === "healthy" || c.status === "running",
          risk_level: "medium" as const,
        })),
      ];

      if (strategies.length === 0) {
        strategies.push({
          name: "agent",
          description: "Core ReAct agent loop",
          enabled: true,
          risk_level: "low" as const,
        });
      }

      sendJson(strategies);
      return;
    }

    // GET /api/cost — inference cost summary
    if (pathname === "/api/cost" && req.method === "GET") {
      let todayCost = 0;
      let todayCalls = 0;
      let totalCost = 0;
      try {
        const tableExists = db.raw
          .prepare(
            "SELECT 1 FROM sqlite_master WHERE type='table' AND name='inference_costs'",
          )
          .get();
        if (tableExists) {
          const today = new Date().toISOString().slice(0, 10);
          const rows = db.raw
            .prepare(
              `SELECT SUM(cost_cents) as total, COUNT(*) as calls
               FROM inference_costs WHERE date(created_at) = ?`,
            )
            .get(today) as { total: number | null; calls: number };
          todayCost = (rows?.total ?? 0) / 100;
          todayCalls = rows?.calls ?? 0;
          const allRows = db.raw
            .prepare("SELECT SUM(cost_cents) as total FROM inference_costs")
            .get() as { total: number | null };
          totalCost = (allRows?.total ?? 0) / 100;
        }
      } catch {
        /* table may not exist */
      }
      sendJson({
        today_cost: todayCost,
        today_calls: todayCalls,
        total_cost: totalCost,
        over_budget: false,
      });
      return;
    }

    // GET /api/risk
    if (pathname === "/api/risk" && req.method === "GET") {
      sendJson({
        risk_level: "LOW",
        daily_loss: 0,
        paused: db.getAgentState() === "sleeping",
      });
      return;
    }

    // POST /api/pause
    if (pathname === "/api/pause" && req.method === "POST") {
      db.setAgentState("sleeping");
      db.setKV("sleep_until", new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toISOString());
      sendJson({ status: "paused" });
      return;
    }

    // POST /api/resume
    if (pathname === "/api/resume" && req.method === "POST") {
      db.deleteKV("sleep_until");
      insertWakeEvent(db.raw, "web", "resume from dashboard");
      sendJson({ status: "resumed" });
      return;
    }

    // POST /api/chat — simple echo/status for now
    if (pathname === "/api/chat" && req.method === "POST") {
      let body = "";
      for await (const chunk of req) body += chunk;
      const payload = JSON.parse(body || "{}");
      const msg = (payload.message || "").toLowerCase();

      if (msg.includes("status")) {
        const state = db.getAgentState();
        sendJson({ response: `System is ${state.toUpperCase()}. Turn count: ${db.getTurnCount()}` });
        return;
      }
      if (msg.includes("help") || msg.includes("帮助")) {
        sendJson({
          response:
            "I can help with:\n1. status — show agent state\n2. strategies — list skills/children\n3. cost — LLM cost summary\n\nTry: 'status' or 'strategies'",
        });
        return;
      }

      sendJson({
        response: `I don't understand '${payload.message || ""}'. Type 'help' for commands.`,
      });
      return;
    }

    res.writeHead(404);
    res.end(JSON.stringify({ error: "Not found" }));
  } catch (err: unknown) {
    logger.error("API error", err instanceof Error ? err : undefined);
    sendJson({ error: String(err) }, 500);
  }
}

function serveFile(res: http.ServerResponse, filePath: string): void {
  const ext = path.extname(filePath);
  const mime = MIME_TYPES[ext] || "application/octet-stream";

  fs.readFile(filePath, (err, data) => {
    if (err) {
      res.writeHead(500);
      res.end("Internal Server Error");
      return;
    }
    res.writeHead(200, { "Content-Type": mime });
    res.end(data);
  });
}

export function startWebServer(
  ctx: WebServerContext,
  port = 8080,
): http.Server {
  const server = createWebServer(ctx);
  server.listen(port, () => {
    logger.info(`Web dashboard: http://localhost:${port}`);
  });
  return server;
}
