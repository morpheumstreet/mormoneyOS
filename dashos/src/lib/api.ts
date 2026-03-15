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

/** Read-only GET helper: parses JSON, handles 404 and error body. */
async function fetchGet<T>(
  path: string,
  options?: { notFoundMsg?: string; parseErrorFromBody?: boolean }
): Promise<T> {
  const res = await fetch(API + path);
  if (!res.ok) {
    if (options?.parseErrorFromBody) {
      const data = await res.json().catch(() => ({}));
      const msg = (data as { error?: string }).error;
      throw new Error(msg || res.statusText);
    }
    const msg = res.status === 404 && options?.notFoundMsg
      ? options.notFoundMsg
      : res.statusText;
    throw new Error(msg);
  }
  return res.json() as Promise<T>;
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

export function putConfig(config: string): Promise<void> {
  return apiFetch("/config", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: config,
  });
}

/** Economic (wallets, USDC balances, treasury policy, constraint mode) */
export interface EconomicAddress {
  address: string;
  chain: string;
  source: string;
}

export interface EconomicBalance {
  address: string;
  chain: string;
  source: string;
  balance: number | null;
  error?: string;
}

export interface TreasuryPolicy {
  maxSingleTransferCents?: number;
  maxHourlyTransferCents?: number;
  maxDailyTransferCents?: number;
  minReserveCents?: number;
  inferenceDailyBudgetCents?: number;
  x402AllowedDomains?: string[];
}

export interface EconomicResponse {
  addresses: EconomicAddress[];
  balances: EconomicBalance[];
  treasuryPolicy: TreasuryPolicy;
  resourceConstraintMode: "auto" | "forced_on" | "forced_off";
}

export function getEconomic(): Promise<EconomicResponse> {
  return fetch(API + "/economic").then((r) => {
    if (!r.ok) throw new Error(r.status === 404 ? "Economic API not available" : r.statusText);
    return r.json();
  });
}

export function putEconomic(
  updates: Partial<{
    treasuryPolicy: TreasuryPolicy;
    resourceConstraintMode: "auto" | "forced_on" | "forced_off";
  }>
): Promise<void> {
  return apiFetch<void>("/economic", {
    method: "PUT",
    body: JSON.stringify(updates),
  });
}

/** Verify a signed message (Ethereum, Solana, Bitcoin, Morpheum) */
export interface VerifyRequest {
  chain: "ethereum" | "solana" | "bitcoin" | "morpheum";
  message: string;
  signature: string;
  address?: string;
  ecPubBytes?: string;
  mldsaPubBytes?: string;
}

export interface VerifyResponse {
  valid: boolean;
  address?: string;
  token?: string;
  error?: string;
}

export function postAuthVerify(req: VerifyRequest): Promise<VerifyResponse> {
  return apiFetch<VerifyResponse>("/auth/verify", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

/** Dev bypass: POST /api/auth/dev-bypass (requires MONEYCLAW_DEV_BYPASS=1). Returns token for agent browser testing. */
export function postAuthDevBypass(): Promise<VerifyResponse> {
  return fetch(API + "/auth/dev-bypass", { method: "POST" }).then((r) => {
    if (!r.ok) throw new Error(r.statusText);
    return r.json();
  });
}

/** Reports (metric snapshots, last_metrics_report) */
export interface ReportSnapshot {
  id: string;
  snapshot_at: string;
  metrics: Record<string, unknown>;
  alerts: unknown[];
}

export interface ReportsResponse {
  last_report?: { status?: string; checkedAt?: string; alerts?: number; error?: string };
  snapshots: ReportSnapshot[];
}

export function getReports(): Promise<ReportsResponse> {
  return fetch(API + "/reports").then((r) => {
    if (!r.ok) throw new Error(r.statusText);
    return r.json();
  });
}

/** Tools (when backend implements GET /api/tools) */
export interface ToolItem {
  name: string;
  description: string;
  enabled: boolean;
}

export interface ToolsResponse {
  tools: ToolItem[];
}

export function getTools(): Promise<ToolsResponse> {
  return fetch(API + "/tools").then((r) => {
    if (!r.ok) throw new Error(r.status === 404 ? "Tools API not available" : r.statusText);
    return r.json();
  });
}

export function patchToolEnabled(name: string, enabled: boolean): Promise<{ name: string; enabled: boolean }> {
  return apiFetch<{ name: string; enabled: boolean }>("/tools/" + encodeURIComponent(name), {
    method: "PATCH",
    body: JSON.stringify({ enabled }),
  });
}

/** Heartbeat schedule (when backend implements GET /api/heartbeat) */
export interface HeartbeatScheduleItem {
  name: string;
  schedule: string;
  task: string;
  enabled: boolean;
  tierMinimum: string;
  lastRun: string;
  nextRun: string;
  leaseUntil: string;
  leaseOwner: string;
}

export interface HeartbeatResponse {
  schedules: HeartbeatScheduleItem[];
}

export function getHeartbeat(): Promise<HeartbeatResponse> {
  return fetch(API + "/heartbeat").then((r) => {
    if (!r.ok) throw new Error(r.status === 404 ? "Heartbeat API not available" : r.statusText);
    return r.json();
  });
}

export function patchHeartbeatEnabled(
  name: string,
  enabled: boolean
): Promise<{ name: string; enabled: boolean }> {
  return apiFetch<{ name: string; enabled: boolean }>(
    "/heartbeat/" + encodeURIComponent(name),
    {
      method: "PATCH",
      body: JSON.stringify({ enabled }),
    }
  );
}

export function patchHeartbeatSchedule(
  name: string,
  schedule: string
): Promise<{ name: string; schedule: string }> {
  return apiFetch<{ name: string; schedule: string }>(
    "/heartbeat/" + encodeURIComponent(name) + "/schedule",
    {
      method: "PATCH",
      body: JSON.stringify({ schedule }),
    }
  );
}

/** Social channels (when backend implements GET /api/social) */
export interface SocialConfigField {
  key: string;
  label: string;
  type: "password" | "text" | "array" | "boolean";
  required: boolean;
  description?: string;
}

export interface SocialChannelItem {
  name: string;
  displayName: string;
  enabled: boolean;
  ready: boolean;
  configFields?: SocialConfigField[];
  config?: Record<string, unknown>;
}

export interface SocialResponse {
  channels: SocialChannelItem[];
}

export function getSocial(): Promise<SocialResponse> {
  return fetch(API + "/social").then((r) => {
    if (!r.ok) throw new Error(r.status === 404 ? "Social API not available" : r.statusText);
    return r.json();
  });
}

export function patchSocialEnabled(name: string, enabled: boolean): Promise<{ name: string; enabled: boolean }> {
  return apiFetch<{ name: string; enabled: boolean }>("/social/" + encodeURIComponent(name), {
    method: "PATCH",
    body: JSON.stringify({ enabled }),
  });
}

export interface PutSocialConfigResponse {
  ok: boolean;
  validated?: boolean;
  enabled?: boolean;
  error?: string;
}

/** Models (LLM providers, model IDs, context limits, cost caps) */
export interface ModelItem {
  id: string;
  provider: string;
  modelId: string;
  apiKeyMasked?: string;
  contextLimit?: number;
  costCapCents?: number;
  priority?: number;
  enabled?: boolean;
}

export interface ModelProvider {
  key: string;
  displayName: string;
  local?: boolean;
  isReseller?: boolean;
  configKey?: string;
  endpointConfigKey?: string;
  endpointValue?: string;
  hasKey?: boolean;
}

/** CanIRun.ai-style catalog entry for model picker */
export interface ModelCatalogEntry {
  provider: string;
  modelId: string;
  displayName: string;
  params: string;
  vramGb: number;
  contextK: number;
  arch: string;
  useCases: string[];
  tier: string;
  description: string;
  isViaReseller?: boolean;
}

export interface ModelsResponse {
  models: ModelItem[];
  providers: ModelProvider[];
  catalog?: ModelCatalogEntry[];
}

export function getModels(): Promise<ModelsResponse> {
  return fetch(API + "/models").then((r) => {
    if (!r.ok) throw new Error(r.status === 404 ? "Models API not available" : r.statusText);
    return r.json();
  });
}

export function postModel(body: {
  provider: string;
  modelId: string;
  apiKey?: string;
  contextLimit?: number;
  costCapCents?: number;
  enabled?: boolean;
}): Promise<ModelItem> {
  return apiFetch<ModelItem>("/models", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function patchModel(
  id: string,
  body: Partial<{
    apiKey: string;
    modelId: string;
    contextLimit: number;
    costCapCents: number;
    enabled: boolean;
  }>
): Promise<ModelItem> {
  return apiFetch<ModelItem>("/models/" + encodeURIComponent(id), {
    method: "PATCH",
    body: JSON.stringify(body),
  });
}

export function deleteModel(id: string): Promise<void> {
  return apiFetch<void>("/models/" + encodeURIComponent(id), {
    method: "DELETE",
  });
}

export function putModelsOrder(ids: string[]): Promise<void> {
  return apiFetch<void>("/models/order", {
    method: "PUT",
    body: JSON.stringify({ ids }),
  });
}

/** Update provider endpoint URL (Ollama, Conway, Azure, Vertex). For local LLMs like Ollama, allows custom base URL. */
export function putProviderEndpoint(
  provider: string,
  url: string
): Promise<{ ok: boolean; provider: string; url: string }> {
  return apiFetch<{ ok: boolean; provider: string; url: string }>(
    "/models/providers/" + encodeURIComponent(provider) + "/endpoint",
    {
      method: "PUT",
      body: JSON.stringify({ url }),
    }
  );
}

/** Soul config (personality, system prompt, tone, behavioral constraints) */
export interface SoulConfig {
  systemPrompt?: string;
  personality?: string;
  tone?: string;
  behavioralConstraints?: string[];
  systemPromptVersions?: string[];
}

export function getSoulConfig(): Promise<SoulConfig> {
  return fetch(API + "/soul/config").then((r) => {
    if (!r.ok) throw new Error(r.status === 404 ? "Soul config API not available" : r.statusText);
    return r.json();
  });
}

export function putSoulConfig(config: Partial<SoulConfig>): Promise<void> {
  return apiFetch<void>("/soul/config", {
    method: "PUT",
    body: JSON.stringify(config),
  });
}

/** Soul enhance: turn casual words into a full system prompt via LLM. Requires auth. */
export interface SoulEnhanceRequest {
  words: string;
  apply?: boolean;
}

export interface SoulEnhanceResponse {
  systemPrompt: string;
}

export function postSoulEnhance(
  words: string,
  apply = false
): Promise<SoulEnhanceResponse> {
  return apiFetch<SoulEnhanceResponse>("/soul/enhance", {
    method: "POST",
    body: JSON.stringify({ words, apply }),
  });
}

/** Tunnel providers (bore, localtunnel, cloudflare, ngrok, tailscale, custom) */
export interface TunnelProviderField {
  name: string;
  type: "password" | "string" | "boolean";
  required: boolean;
  label?: string;
  help?: string;
}

export interface TunnelProviderSchema {
  fields: TunnelProviderField[];
}

export interface TunnelProvidersResponse {
  providers: string[];
  schemas: Record<string, TunnelProviderSchema>;
  config: {
    defaultProvider: string;
    providers: Record<
      string,
      {
        enabled?: boolean;
        token?: string;
        authToken?: string;
        authKey?: string;
        domain?: string;
        hostname?: string;
        funnel?: boolean;
        startCommand?: string;
        urlPattern?: string;
      }
    >;
  };
}

export interface TunnelItem {
  port: number;
  provider: string;
  public_url: string;
}

export interface TunnelsResponse {
  tunnels: TunnelItem[];
}

export function getTunnelProviders(): Promise<TunnelProvidersResponse> {
  return fetch(API + "/tunnels/providers").then((r) => {
    if (!r.ok) throw new Error(r.status === 404 ? "Tunnel API not available" : r.statusText);
    return r.json();
  });
}

export function getTunnels(): Promise<TunnelsResponse> {
  return fetch(API + "/tunnels").then((r) => {
    if (!r.ok) throw new Error(r.status === 404 ? "Tunnel API not available" : r.statusText);
    return r.json();
  });
}

export function putTunnelProvider(
  name: string,
  body: Record<string, unknown>
): Promise<{ ok: boolean; provider: string }> {
  return apiFetch<{ ok: boolean; provider: string }>(
    "/tunnels/providers/" + encodeURIComponent(name),
    {
      method: "PUT",
      body: JSON.stringify(body),
    }
  );
}

export function postTunnelProviderRestart(
  name: string
): Promise<{ ok: boolean; provider: string; restarted: boolean }> {
  return apiFetch<{ ok: boolean; provider: string; restarted: boolean }>(
    "/tunnels/providers/" + encodeURIComponent(name) + "/restart",
    { method: "POST" }
  );
}

/** Skills (installed + ClawHub discovery) */
export interface SkillItem {
  name: string;
  description?: string;
  source: string;
  path?: string;
  enabled: boolean;
  trusted?: boolean;
  auto_activate?: number;
}

export interface SkillsResponse {
  skills: SkillItem[];
}

export interface DiscoveryResult {
  slug: string;
  displayName?: string;
  summary?: string;
  version?: string;
  score?: number;
}

export interface DiscoverySearchResponse {
  results: DiscoveryResult[];
}

export interface DiscoveryListResponse {
  items: DiscoveryResult[];
  nextCursor?: string;
}

export interface RecommendedSkill {
  slug: string;
  displayName?: string;
  summary?: string;
  version?: string;
  installed?: boolean;
}

export interface RecommendedResponse {
  recommended: RecommendedSkill[];
}

export function getSkills(params?: {
  filter?: "all" | "enabled" | "disabled";
  trusted?: "all" | "trusted" | "untrusted";
}): Promise<SkillsResponse> {
  const search = new URLSearchParams();
  if (params?.filter) search.set("filter", params.filter);
  if (params?.trusted) search.set("trusted", params.trusted);
  const qs = search.toString();
  return fetch(`${API}/skills${qs ? `?${qs}` : ""}`).then((r) => {
    if (!r.ok) throw new Error(r.status === 404 ? "Skills API not available" : r.statusText);
    return r.json();
  });
}

export function getSkill(name: string): Promise<SkillItem> {
  return fetch(API + "/skills/" + encodeURIComponent(name)).then((r) => {
    if (!r.ok) throw new Error(r.status === 404 ? "Skill not found" : r.statusText);
    return r.json();
  });
}

export function getSkillsDiscovery(params?: {
  q?: string;
  limit?: number;
  cursor?: string;
}): Promise<DiscoverySearchResponse | DiscoveryListResponse> {
  const search = new URLSearchParams();
  if (params?.q) search.set("q", params.q);
  if (params?.limit != null) search.set("limit", String(params.limit));
  if (params?.cursor) search.set("cursor", params.cursor);
  const qs = search.toString();
  return fetch(`${API}/skills/discovery${qs ? `?${qs}` : ""}`).then((r) => {
    if (!r.ok) throw new Error(r.status === 404 ? "Discovery API not available" : r.statusText);
    return r.json();
  });
}

export function getSkillsRecommended(): Promise<RecommendedResponse> {
  return fetch(API + "/skills/recommended").then((r) => {
    if (!r.ok) throw new Error(r.status === 404 ? "Recommended API not available" : r.statusText);
    return r.json();
  });
}

export function postSkillInstall(body: {
  source: "clawhub";
  id: string;
  version?: string;
  name?: string;
  description?: string;
} | {
  name: string;
  path: string;
  description?: string;
}): Promise<{ name: string; source?: string; path?: string; enabled?: boolean }> {
  return apiFetch<{ name: string; source?: string; path?: string; enabled?: boolean }>("/skills", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function patchSkill(name: string, body: Partial<{ enabled: boolean; description: string; instructions: string }>): Promise<SkillItem> {
  return apiFetch<SkillItem>("/skills/" + encodeURIComponent(name), {
    method: "PATCH",
    body: JSON.stringify(body),
  });
}

export function deleteSkill(name: string): Promise<void> {
  return apiFetch<void>("/skills/" + encodeURIComponent(name), { method: "DELETE" });
}

export function patchSkillActivate(name: string): Promise<{ name: string; enabled: boolean }> {
  return apiFetch<{ name: string; enabled: boolean }>("/skills/" + encodeURIComponent(name) + "/activate", {
    method: "PATCH",
  });
}

export function patchSkillDeactivate(name: string): Promise<{ name: string; enabled: boolean }> {
  return apiFetch<{ name: string; enabled: boolean }>("/skills/" + encodeURIComponent(name) + "/deactivate", {
    method: "PATCH",
  });
}

/** Wallet (mnemonic-derived, multi-chain; no mnemonic/keys exposed) */
const WALLET_PATH = "/wallet";

export interface WalletResponse {
  exists: boolean;
  currentIndex?: number;
  address?: string;
  defaultChain?: string;
  wordCount?: number;
  error?: string;
}

export interface WalletAddressResponse {
  chain: string;
  index: number;
  address: string;
}

export interface WalletRotateResponse {
  currentIndex: number;
  targetIndex: number;
  currentAddresses: Record<string, string>;
  newAddresses: Record<string, string>;
  preview?: boolean;
  confirmed?: boolean;
  message?: string;
}

/** Identity labels (HD index → friendly name) stored in automaton.json */
export interface WalletIdentityLabelsResponse {
  identityLabels: Record<string, string>;
}

export interface WalletIdentityLabelsWriteResponse extends WalletIdentityLabelsResponse {
  ok: boolean;
}

/** Single label update: set or remove (empty string) */
export interface WalletIdentityLabelPatchSingle {
  index: number;
  label: string;
}

/** Merge multiple labels into existing */
export interface WalletIdentityLabelPatchMerge {
  identityLabels: Record<string, string>;
}

export type WalletIdentityLabelPatchBody =
  | WalletIdentityLabelPatchSingle
  | WalletIdentityLabelPatchMerge;

export function getWallet(): Promise<WalletResponse> {
  return fetchGet<WalletResponse>(WALLET_PATH, { notFoundMsg: "Wallet API not available" });
}

export function getWalletAddress(chain: string, index?: number): Promise<WalletAddressResponse> {
  const params = new URLSearchParams({ chain });
  if (index != null && index > 0) params.set("index", String(index));
  return fetchGet<WalletAddressResponse>(`${WALLET_PATH}/address?${params}`, {
    notFoundMsg: "Wallet address API not available",
    parseErrorFromBody: true,
  });
}

export function postWalletRotate(body: {
  toIndex: number;
  preview?: boolean;
  confirm?: boolean;
}): Promise<WalletRotateResponse> {
  return apiFetch<WalletRotateResponse>(`${WALLET_PATH}/rotate`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function postWalletClearCache(): Promise<{ ok: boolean; message?: string }> {
  return apiFetch<{ ok: boolean; message?: string }>(`${WALLET_PATH}/clear-cache`, {
    method: "POST",
  });
}

export function getWalletIdentityLabels(): Promise<WalletIdentityLabelsResponse> {
  return fetchGet<WalletIdentityLabelsResponse>(`${WALLET_PATH}/identity-labels`, {
    notFoundMsg: "Wallet identity labels API not available",
  });
}

export function putWalletIdentityLabels(
  identityLabels: Record<string, string>
): Promise<WalletIdentityLabelsWriteResponse> {
  return apiFetch<WalletIdentityLabelsWriteResponse>(`${WALLET_PATH}/identity-labels`, {
    method: "PUT",
    body: JSON.stringify({ identityLabels }),
  });
}

export function patchWalletIdentityLabels(
  body: WalletIdentityLabelPatchBody
): Promise<WalletIdentityLabelsWriteResponse> {
  return apiFetch<WalletIdentityLabelsWriteResponse>(`${WALLET_PATH}/identity-labels`, {
    method: "PATCH",
    body: JSON.stringify(body),
  });
}

export async function putSocialConfig(
  name: string,
  config: Record<string, unknown>
): Promise<PutSocialConfigResponse> {
  const token = getToken();
  const headers: HeadersInit = { "Content-Type": "application/json" };
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const res = await fetch(API + "/social/" + encodeURIComponent(name) + "/config", {
    method: "PUT",
    headers,
    body: JSON.stringify(config),
  });

  const data = (await res.json().catch(() => ({}))) as PutSocialConfigResponse;
  if (res.status === 401) {
    window.dispatchEvent(new Event("dashos-unauthorized"));
    throw new UnauthorizedError();
  }
  if (!res.ok) {
    data.ok = false;
    data.validated = false;
    data.error = data.error || `Request failed: ${res.status}`;
  }
  return data;
}
