import { useEffect, useState, useCallback } from "react";
import {
  getEconomic,
  putEconomic,
  type EconomicResponse,
  type TreasuryPolicy,
} from "@/lib/api";

type ResourceMode = "auto" | "forced_on" | "forced_off";

function extractErrorMessage(e: unknown): string {
  return e instanceof Error ? e.message : "Unknown error";
}

export function useEconomicConfig(hasWriteAccess: boolean) {
  const [data, setData] = useState<EconomicResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [resourceMode, setResourceMode] = useState<ResourceMode>("auto");
  const [treasury, setTreasury] = useState<TreasuryPolicy>({});

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await getEconomic();
      setData(res);
      setResourceMode(res.resourceConstraintMode || "auto");
      setTreasury(res.treasuryPolicy || {});
    } catch (e) {
      setError(extractErrorMessage(e) || "Load failed");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleSave = useCallback(async () => {
    if (!hasWriteAccess) {
      setError("Write access required. Connect wallet and sign.");
      return;
    }
    setSaving(true);
    setError(null);
    setSuccess(null);
    try {
      await putEconomic({
        resourceConstraintMode: resourceMode,
        treasuryPolicy: treasury,
      });
      setSuccess("Economic settings saved.");
    } catch (e) {
      setError(extractErrorMessage(e) || "Save failed");
    } finally {
      setSaving(false);
    }
  }, [hasWriteAccess, resourceMode, treasury]);

  const updateTreasuryField = useCallback(
    <K extends keyof TreasuryPolicy>(key: K, value: number) => {
      setTreasury((t) => ({ ...t, [key]: value }));
    },
    []
  );

  return {
    data,
    loading,
    saving,
    error,
    success,
    resourceMode,
    setResourceMode,
    treasury,
    updateTreasuryField,
    handleSave,
  };
}
