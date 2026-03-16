import { Search, Cloud, Server } from "lucide-react";
import { FilterButton } from "@/components/ui/FilterButton";
import { inputMd } from "@/lib/theme";
import type { ModelCatalogEntry, ModelProvider } from "@/lib/api";
import { TIER_COLORS } from "./constants";

const TIER_OPTIONS = [
  { key: "all", label: "All" },
  { key: "sab", label: "Can run (S/A/B)" },
  { key: "cd", label: "Tight fit (C/D)" },
  { key: "f", label: "Too heavy (F)" },
];

const USE_CASE_OPTIONS = [
  { key: "all", label: "All" },
  { key: "chat", label: "Chat" },
  { key: "code", label: "Code" },
  { key: "reasoning", label: "Reasoning" },
  { key: "vision", label: "Vision" },
];

interface ModelCatalogProps {
  hasWriteAccess: boolean;
  providers: ModelProvider[];
  filteredCatalog: ModelCatalogEntry[];
  catalogType: "cloud" | "local";
  setCatalogType: (t: "cloud" | "local") => void;
  catalogQuery: string;
  setCatalogQuery: (q: string) => void;
  catalogTierFilter: string;
  setCatalogTierFilter: (t: string) => void;
  catalogUseCaseFilter: string;
  setCatalogUseCaseFilter: (u: string) => void;
  catalogSort: string;
  setCatalogSort: (s: string) => void;
  pickFromCatalog: (entry: ModelCatalogEntry) => void;
}

export function ModelCatalog({
  hasWriteAccess,
  providers,
  filteredCatalog,
  catalogType,
  setCatalogType,
  catalogQuery,
  setCatalogQuery,
  catalogTierFilter,
  setCatalogTierFilter,
  catalogUseCaseFilter,
  setCatalogUseCaseFilter,
  catalogSort,
  setCatalogSort,
  pickFromCatalog,
}: ModelCatalogProps) {
  return (
    <div className="electric-card p-4">
      <h3 className="text-sm font-medium text-white mb-1">Model catalog</h3>
      <p className="text-xs text-[#8aa8df] mb-3">
        {catalogType === "cloud"
          ? "Public API models from OpenAI, xAI, Qwen, etc. "
          : "Local models (Ollama, LocalAI, llama.cpp, LM Studio, vLLM, Jan AI). "}
        {hasWriteAccess
          ? "Click to fill the Add Model form."
          : "Connect wallet to add models."}{" "}
        <a
          href="https://llm-stats.com/"
          target="_blank"
          rel="noopener noreferrer"
          className="text-[#9bc3ff] hover:underline"
        >
          llm-stats.com
        </a>
        {" · "}
        <a
          href="https://www.canirun.ai/"
          target="_blank"
          rel="noopener noreferrer"
          className="text-[#9bc3ff] hover:underline"
        >
          CanIRun.ai
        </a>
      </p>
      <div className="flex gap-2 mb-3">
        <FilterButton
          active={catalogType === "cloud"}
          onClick={() => setCatalogType("cloud")}
          size="lg"
        >
          <Cloud className="h-4 w-4" />
          API / Cloud
        </FilterButton>
        <FilterButton
          active={catalogType === "local"}
          onClick={() => setCatalogType("local")}
          size="lg"
        >
          <Server className="h-4 w-4" />
          Local
        </FilterButton>
      </div>
      <div className="relative mb-3">
        <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-[#6b8fcc]" />
        <input
          type="text"
          value={catalogQuery}
          onChange={(e) => setCatalogQuery(e.target.value)}
          placeholder="Search by name, provider, use case…"
          className={`w-full pl-8 pr-3 py-2 rounded-lg ${inputMd}`}
        />
      </div>
      <div className="flex flex-wrap items-center gap-2 mb-3">
        <span className="text-xs text-[#6b8fcc]">Tier:</span>
        {TIER_OPTIONS.map(({ key, label }) => (
          <FilterButton
            key={key}
            active={catalogTierFilter === key}
            onClick={() => setCatalogTierFilter(key)}
          >
            {label}
          </FilterButton>
        ))}
        <span className="text-xs text-[#6b8fcc] ml-2">Tasks:</span>
        {USE_CASE_OPTIONS.map(({ key, label }) => (
          <FilterButton
            key={key}
            active={catalogUseCaseFilter === key}
            onClick={() => setCatalogUseCaseFilter(key)}
          >
            {label}
          </FilterButton>
        ))}
        <span className="text-xs text-[#6b8fcc] ml-2">Sort:</span>
        <select
          value={catalogSort}
          onChange={(e) => setCatalogSort(e.target.value)}
          className={`px-2.5 py-1 rounded text-xs ${inputMd}`}
        >
          <option value="params">Params ↓</option>
          <option value="context">Context ↓</option>
          <option value="vram">VRAM ↓</option>
        </select>
      </div>
      <div className="max-h-80 overflow-y-auto rounded-lg border border-[#29509c] bg-[#071228]/50">
        <div className="divide-y divide-[#1a3670]/80">
          {filteredCatalog.map((entry) => {
            const prov = providers.find((p) => p.key === entry.provider);
            const providerName = prov?.displayName ?? entry.provider;
            const hasKey = prov?.hasKey ?? false;
            const isCloud = entry.vramGb === 0;
            const tierCls =
              TIER_COLORS[entry.tier] ?? "bg-[#29509c]/50 text-[#8aa8df]";
            return (
              <button
                key={`${entry.provider}:${entry.modelId}`}
                type="button"
                onClick={() => hasWriteAccess && pickFromCatalog(entry)}
                disabled={!hasWriteAccess}
                className={`w-full text-left px-3 py-2.5 transition-colors flex items-start gap-3 ${
                  hasWriteAccess
                    ? "hover:bg-[#07132f]/70 cursor-pointer"
                    : "cursor-default opacity-90"
                }`}
              >
                <span
                  className={`shrink-0 w-6 h-6 rounded text-xs font-bold flex items-center justify-center border ${tierCls}`}
                  title="Tier"
                >
                  {entry.tier}
                </span>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="font-medium text-white text-sm">
                      {entry.displayName}
                    </span>
                    {entry.params && (
                      <span className="text-xs text-[#6b8fcc]">
                        {entry.params}
                      </span>
                    )}
                    <span className="text-xs text-[#8aa8df]">
                      {providerName}
                    </span>
                    {isCloud && hasKey && (
                      <span className="text-[10px] px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-300 border border-emerald-500/40">
                        Available
                      </span>
                    )}
                    {isCloud && !hasKey && prov?.configKey && (
                      <span className="text-[10px] px-1.5 py-0.5 rounded bg-amber-500/20 text-amber-300 border border-amber-500/40">
                        Add API key
                      </span>
                    )}
                    {entry.isViaReseller && (
                      <span
                        className="text-[10px] px-1.5 py-0.5 rounded bg-slate-500/20 text-slate-300 border border-slate-500/40"
                        title="Via reseller (aggregates models from other developers)"
                      >
                        via Reseller
                      </span>
                    )}
                  </div>
                  <div className="mt-0.5 flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-[#6b8fcc]">
                    {entry.vramGb > 0 && (
                      <span>{entry.vramGb} GB VRAM</span>
                    )}
                    <span>{entry.contextK}K ctx</span>
                    <span>{entry.arch}</span>
                    {entry.useCases.length > 0 && (
                      <span>{entry.useCases.slice(0, 3).join(", ")}</span>
                    )}
                  </div>
                  {entry.description && (
                    <p className="mt-0.5 text-xs text-[#8aa8df]/80 line-clamp-1">
                      {entry.description}
                    </p>
                  )}
                </div>
              </button>
            );
          })}
        </div>
        {filteredCatalog.length === 0 && (
          <div className="px-4 py-6 text-center text-sm text-[#6b8fcc]">
            No models match your filters.
          </div>
        )}
      </div>
    </div>
  );
}
