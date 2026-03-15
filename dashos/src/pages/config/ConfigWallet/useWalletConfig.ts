import { useState, useCallback } from "react";
import {
  getWallet,
  getWalletAddress,
  postWalletRotate,
  getConfig,
  putConfig,
} from "@/lib/api";
import type { WalletResponse, WalletAddressResponse } from "@/lib/api";
import { SUPPORTED_CHAINS, DEFAULT_CHAIN } from "./constants";

export interface ChainAddress {
  chain: string;
  name: string;
  caip2: string;
  address: string;
  error?: string;
  isMorpheum?: boolean;
}

export function useWalletConfig() {
  const [wallet, setWallet] = useState<WalletResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const loadWallet = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const w = await getWallet();
      setWallet(w);
      return w;
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load wallet");
      setWallet(null);
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  const deriveAddressesForIndex = useCallback(
    async (index: number): Promise<ChainAddress[]> => {
      const results: ChainAddress[] = [];
      for (const ch of SUPPORTED_CHAINS) {
        try {
          const res = await getWalletAddress(ch.caip2, index);
          results.push({
            chain: ch.name,
            caip2: ch.caip2,
            name: ch.name,
            address: res.address,
            isMorpheum: ch.isMorpheum,
          });
        } catch (e) {
          results.push({
            chain: ch.name,
            caip2: ch.caip2,
            name: ch.name,
            address: "",
            error: e instanceof Error ? e.message : "Derivation failed",
            isMorpheum: ch.isMorpheum,
          });
        }
      }
      return results;
    },
    []
  );

  const rotateToIndex = useCallback(
    async (toIndex: number, confirm: boolean) => {
      setError(null);
      try {
        return await postWalletRotate({
          toIndex,
          preview: !confirm,
          confirm,
        });
      } catch (e) {
        setError(e instanceof Error ? e.message : "Rotate failed");
        throw e;
      }
    },
    []
  );

  const updateConfigIdentity = useCallback(
    async (updates: { defaultChain?: string; identityLabel?: string; index?: number }) => {
      let raw: string;
      try {
        raw = await getConfig();
      } catch {
        raw = "{}";
      }
      let obj: Record<string, unknown>;
      try {
        obj = typeof raw === "string" ? JSON.parse(raw || "{}") : (raw as Record<string, unknown>);
      } catch {
        obj = {};
      }
      if (updates.defaultChain) obj.defaultChain = updates.defaultChain;
      if (updates.identityLabel != null && updates.index != null) {
        const labels = (obj.identityLabels as Record<string, string>) || {};
        labels[String(updates.index)] = updates.identityLabel;
        obj.identityLabels = labels;
      }
      await putConfig(JSON.stringify(obj, null, 2));
    },
    []
  );

  const getIdentityLabel = useCallback(async (index: number): Promise<string> => {
    try {
      const raw = await getConfig();
      const obj = typeof raw === "string" ? JSON.parse(raw || "{}") : (raw as Record<string, unknown>);
      const labels = obj.identityLabels as Record<string, string> | undefined;
      return labels?.[String(index)] ?? "";
    } catch {
      return "";
    }
  }, []);

  return {
    wallet,
    loading,
    error,
    success,
    setError,
    setSuccess,
    loadWallet,
    deriveAddressesForIndex,
    rotateToIndex,
    updateConfigIdentity,
    getIdentityLabel,
  };
}
