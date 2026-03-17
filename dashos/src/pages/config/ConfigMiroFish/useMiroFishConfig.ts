import { useEffect, useState, useCallback } from "react";
import {
  getMiroFishConfig,
  putMiroFishConfig,
  type MiroFishConfig,
} from "@/lib/api";

function extractErrorMessage(e: unknown): string {
  return e instanceof Error ? e.message : "Unknown error";
}

const DEFAULT_CONFIG: MiroFishConfig = {
  enabled: false,
  base_url: "http://localhost:5001",
  timeout_seconds: 300,
  default_llm: "qwen-plus",
  max_agents: 2000,
};

export function useMiroFishConfig(hasWriteAccess: boolean) {
  const [config, setConfig] = useState<MiroFishConfig>(DEFAULT_CONFIG);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await getMiroFishConfig();
      setConfig({
        enabled: res.enabled ?? false,
        base_url: res.base_url ?? "http://localhost:5001",
        timeout_seconds: res.timeout_seconds ?? 300,
        default_llm: res.default_llm ?? "qwen-plus",
        max_agents: res.max_agents ?? 2000,
      });
    } catch (e) {
      setError(extractErrorMessage(e) || "Load failed");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchConfig();
  }, [fetchConfig]);

  const handleSave = useCallback(async () => {
    if (!hasWriteAccess) {
      setError("Write access required. Connect wallet and sign.");
      return;
    }
    setSaving(true);
    setError(null);
    setSuccess(null);
    try {
      await putMiroFishConfig(config);
      setSuccess("MiroFish config saved.");
    } catch (e) {
      setError(extractErrorMessage(e) || "Save failed");
    } finally {
      setSaving(false);
    }
  }, [hasWriteAccess, config]);

  const updateConfig = useCallback(<K extends keyof MiroFishConfig>(
    key: K,
    value: MiroFishConfig[K]
  ) => {
    setConfig((c) => ({ ...c, [key]: value }));
  }, []);

  return {
    config,
    loading,
    saving,
    error,
    success,
    updateConfig,
    handleSave,
  };
}
