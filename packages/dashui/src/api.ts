const API_BASE = "";

export interface Status {
  is_running: boolean;
  state: string;
  tick_count: number;
  wallet_value: number;
  today_pnl: number;
  dry_run: boolean;
  address: string;
  name: string;
  version: string;
}

export interface Strategy {
  name: string;
  description: string;
  enabled: boolean;
  risk_level: string;
}

export interface Cost {
  today_cost: number;
  today_calls: number;
  total_cost: number;
  over_budget: boolean;
}

export interface Risk {
  risk_level: string;
  daily_loss: number;
  paused: boolean;
}

export interface ChatResponse {
  response: string;
}

export async function getStatus(): Promise<Status> {
  const res = await fetch(`${API_BASE}/api/status`);
  return res.json();
}

export async function getStrategies(): Promise<Strategy[]> {
  const res = await fetch(`${API_BASE}/api/strategies`);
  return res.json();
}

export async function getCost(): Promise<Cost> {
  const res = await fetch(`${API_BASE}/api/cost`);
  return res.json();
}

export async function getRisk(): Promise<Risk> {
  const res = await fetch(`${API_BASE}/api/risk`);
  return res.json();
}

export async function postPause(): Promise<{ status: string }> {
  const res = await fetch(`${API_BASE}/api/pause`, { method: "POST" });
  return res.json();
}

export async function postResume(): Promise<{ status: string }> {
  const res = await fetch(`${API_BASE}/api/resume`, { method: "POST" });
  return res.json();
}

export async function postChat(message: string): Promise<ChatResponse> {
  const res = await fetch(`${API_BASE}/api/chat`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message }),
  });
  return res.json();
}
