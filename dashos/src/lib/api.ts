import { getToken } from "./auth";

const API = "/api";

export interface StatusResponse {
  agent_state?: string;
  today_pnl?: number;
  paused?: boolean;
  tick?: number;
  name?: string;
  address?: string;
  chain?: string;
}

export interface Strategy {
  name: string;
  risk_level?: string;
}

export interface CostResponse {
  today_cost?: number;
  today_calls?: number;
  total_cost?: number;
  over_budget?: boolean;
}

export interface RiskResponse {
  risk_level?: string;
  paused?: boolean;
  daily_loss?: number;
}

export class UnauthorizedError extends Error {
  constructor() {
    super("Unauthorized");
    this.name = "UnauthorizedError";
  }
}

async function apiFetch<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const token = getToken();
  const headers = new Headers(options.headers);

  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  if (
    options.body &&
    typeof options.body === "string" &&
    !headers.has("Content-Type")
  ) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(API + path, { ...options, headers });

  if (response.status === 401) {
    window.dispatchEvent(new Event("dashos-unauthorized"));
    throw new UnauthorizedError();
  }

  if (!response.ok) {
    const text = await response.text().catch(() => "");
    throw new Error(`API ${response.status}: ${text || response.statusText}`);
  }

  if (response.status === 204) {
    return undefined as unknown as T;
  }

  return response.json() as Promise<T>;
}

/** Read-only endpoints (no auth required by mormoneyOS) */
export function getStatus(): Promise<StatusResponse> {
  return fetch(API + "/status").then((r) => {
    if (!r.ok) throw new Error(r.statusText);
    return r.json();
  });
}

export function getStrategies(): Promise<Strategy[]> {
  return fetch(API + "/strategies").then((r) => {
    if (!r.ok) throw new Error(r.statusText);
    return r.json();
  });
}

export function getCost(): Promise<CostResponse> {
  return fetch(API + "/cost").then((r) => {
    if (!r.ok) throw new Error(r.statusText);
    return r.json();
  });
}

export function getRisk(): Promise<RiskResponse> {
  return fetch(API + "/risk").then((r) => {
    if (!r.ok) throw new Error(r.statusText);
    return r.json();
  });
}

/** Write endpoints (use bearer token when available) */
export function postPause(): Promise<{ status?: string }> {
  return apiFetch<{ status?: string }>("/pause", { method: "POST" });
}

export function postResume(): Promise<{ status?: string }> {
  return apiFetch<{ status?: string }>("/resume", { method: "POST" });
}

export function postChat(message: string): Promise<{ response?: string }> {
  return apiFetch<{ response?: string }>("/chat", {
    method: "POST",
    body: JSON.stringify({ message }),
  });
}

/** Config (when backend implements GET/PUT /api/config) */
export function getConfig(): Promise<string> {
  return fetch(API + "/config")
    .then((r) => {
      if (!r.ok) throw new Error(r.status === 404 ? "Config API not yet available" : r.statusText);
      return r.text();
    })
    .then((text) => {
      try {
        const data = JSON.parse(text);
        return typeof data.content === "string" ? data.content : data;
      } catch {
        return text;
      }
    });
}

export function putConfig(toml: string): Promise<void> {
  return apiFetch("/config", {
    method: "PUT",
    headers: { "Content-Type": "application/toml" },
    body: toml,
  });
}
