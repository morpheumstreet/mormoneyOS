import { useEffect, useState, useCallback } from "react";
import {
  getAuthConfig,
  putAuthConfig,
  type AuthConfig,
} from "@/lib/api";

function extractErrorMessage(e: unknown): string {
  return e instanceof Error ? e.message : "Unknown error";
}

export function useAuthConfig(hasWriteAccess: boolean) {
  const [config, setConfig] = useState<AuthConfig>({ creator_address: "" });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await getAuthConfig();
      setConfig({
        creator_address: res.creator_address ?? "",
        guest_access_enabled: res.guest_access_enabled ?? false,
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
      const res = await putAuthConfig({
        creator_address: config.creator_address,
        guest_access_enabled: config.guest_access_enabled ?? false,
      });
      setConfig({
        creator_address: res.creator_address ?? "",
        guest_access_enabled: res.guest_access_enabled ?? false,
      });
      setSuccess("Auth config saved.");
    } catch (e) {
      setError(extractErrorMessage(e) || "Save failed");
    } finally {
      setSaving(false);
    }
  }, [hasWriteAccess, config]);

  const updateConfig = useCallback(<K extends keyof AuthConfig>(
    key: K,
    value: AuthConfig[K]
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
