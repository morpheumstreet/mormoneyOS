import type { ModelCatalogEntry, ModelProvider } from "@/lib/api";

export const DEFAULT_CONTEXT_LIMIT = 8192;

/** Providers eligible for Add model: local (e.g. Ollama) or have API key configured. */
export function eligibleProviders(providers: ModelProvider[]): ModelProvider[] {
  return providers.filter((p) => p.local || p.hasKey);
}
export const DEFAULT_COST_CAP_CENTS = 500;

/** Fallback catalog when API returns empty — ensures CanIRun.ai design always shows */
export const FALLBACK_CATALOG: ModelCatalogEntry[] = [
  {
    provider: "openai",
    modelId: "gpt-4o",
    displayName: "GPT-4o",
    params: "",
    vramGb: 0,
    contextK: 128,
    arch: "Dense",
    useCases: ["chat", "code", "vision"],
    tier: "S",
    description: "OpenAI flagship",
  },
  {
    provider: "openai",
    modelId: "gpt-4o-mini",
    displayName: "GPT-4o Mini",
    params: "",
    vramGb: 0,
    contextK: 128,
    arch: "Dense",
    useCases: ["chat", "code"],
    tier: "S",
    description: "Fast and capable",
  },
  {
    provider: "groq",
    modelId: "llama-3.3-70b-versatile",
    displayName: "Llama 3.3 70B",
    params: "70B",
    vramGb: 0,
    contextK: 128,
    arch: "Dense",
    useCases: ["chat", "code", "reasoning"],
    tier: "S",
    description: "Best open at 70B (Groq)",
  },
  {
    provider: "deepseek",
    modelId: "deepseek-chat",
    displayName: "DeepSeek Chat",
    params: "",
    vramGb: 0,
    contextK: 64,
    arch: "Dense",
    useCases: ["chat", "code"],
    tier: "S",
    description: "Strong coding",
  },
  {
    provider: "ollama",
    modelId: "llama3.1",
    displayName: "Llama 3.1 8B",
    params: "8B",
    vramGb: 4.1,
    contextK: 128,
    arch: "Dense",
    useCases: ["chat", "code", "reasoning"],
    tier: "S",
    description: "Great quality/speed",
  },
  {
    provider: "ollama",
    modelId: "qwen2.5",
    displayName: "Qwen 2.5 7B",
    params: "7B",
    vramGb: 3.6,
    contextK: 128,
    arch: "Dense",
    useCases: ["chat", "code"],
    tier: "S",
    description: "Strong multilingual",
  },
  {
    provider: "chatjimmy",
    modelId: "llama3.1-8B",
    displayName: "Llama 3.1 8B (ChatJimmy)",
    params: "8B",
    vramGb: 0,
    contextK: 8,
    arch: "Dense",
    useCases: ["chat", "code"],
    tier: "A",
    description: "Free, no API key",
  },
];

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
