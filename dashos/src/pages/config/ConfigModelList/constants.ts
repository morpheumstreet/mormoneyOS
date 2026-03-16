import type { ModelProvider } from "@/lib/api";

export const DEFAULT_CONTEXT_LIMIT = 8192;

export function getEndpointConfigKey(
  provider: string,
  providers: ModelProvider[]
): string | undefined {
  return providers.find((p) => p.key === provider)?.endpointConfigKey;
}

/** Providers eligible for Add model: local (e.g. Ollama) or have API key configured. */
export function eligibleProviders(providers: ModelProvider[]): ModelProvider[] {
  return providers.filter((p) => p.local || p.hasKey);
}
export const DEFAULT_COST_CAP_CENTS = 500;

/** Local providers (Ollama, LocalAI, etc.) — used for Add form UI and catalog pick */
export const LOCAL_PROVIDERS = [
  "ollama",
  "localai",
  "llamacpp",
  "lmstudio",
  "vllm",
  "janai",
  "g4f",
] as const;

/** Default endpoint URLs for local providers */
export const DEFAULT_LOCAL_URLS: Record<string, string> = {
  ollama: "http://localhost:11434",
  localai: "http://localhost:8080",
  llamacpp: "http://localhost:8080",
  lmstudio: "http://localhost:1234",
  vllm: "http://localhost:8000",
  janai: "http://localhost:1337",
  g4f: "http://localhost:13145",
};

export const TIER_SETS: Record<string, string[]> = {
  sab: ["S", "A", "B"],
  cd: ["C", "D"],
  f: ["F"],
};

export const TIER_COLORS: Record<string, string> = {
  S: "bg-emerald-500/20 text-emerald-300 border-emerald-500/40",
  A: "bg-blue-500/20 text-blue-300 border-blue-500/40",
  B: "bg-cyan-500/20 text-cyan-300 border-cyan-500/40",
  C: "bg-amber-500/20 text-amber-300 border-amber-500/40",
  D: "bg-orange-500/20 text-orange-300 border-orange-500/40",
  F: "bg-rose-500/20 text-rose-300 border-rose-500/40",
};
