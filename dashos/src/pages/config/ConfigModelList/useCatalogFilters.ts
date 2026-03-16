import { useMemo } from "react";
import type { ModelCatalogEntry } from "@/lib/api";
import { TIER_SETS } from "./constants";

export function useCatalogFilters(
  catalog: ModelCatalogEntry[],
  filters: {
    type: "cloud" | "local";
    query: string;
    tier: string;
    useCase: string;
    sort: string;
  }
) {
  return useMemo(() => {
    const displayCatalog = catalog;
    let result = displayCatalog;

    if (filters.type === "cloud") {
      result = result.filter((c) => c.vramGb === 0);
    } else {
      result = result.filter((c) => c.vramGb > 0);
    }

    if (filters.query.trim()) {
      const q = filters.query.toLowerCase();
      result = result.filter(
        (c) =>
          c.displayName.toLowerCase().includes(q) ||
          c.modelId.toLowerCase().includes(q) ||
          c.provider.toLowerCase().includes(q) ||
          c.description.toLowerCase().includes(q) ||
          c.useCases.some((u) => u.toLowerCase().includes(q))
      );
    }

    if (filters.tier !== "all") {
      const allowed = TIER_SETS[filters.tier] ?? [];
      result = result.filter((c) => allowed.includes(c.tier));
    }

    if (filters.useCase !== "all") {
      result = result.filter((c) =>
        c.useCases.some(
          (u) => u.toLowerCase() === filters.useCase.toLowerCase()
        )
      );
    }

    result = [...result].sort((a, b) => {
      if (filters.sort === "params") {
        const parseParams = (p: string) =>
          parseFloat(p.replace(/[^\d.]/g, "")) || 0;
        return parseParams(b.params) - parseParams(a.params);
      }
      if (filters.sort === "context") return b.contextK - a.contextK;
      if (filters.sort === "vram") return b.vramGb - a.vramGb;
      return 0;
    });

    return result;
  }, [catalog, filters]);
}
