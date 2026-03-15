import { useState, useCallback } from "react";
import {
  getWallet,
  getWalletAddress,
  postWalletRotate,
  getWalletIdentityLabels,
  patchWalletIdentityLabels,
  getConfig,
  putConfig,
} from "@/lib/api";
import type { WalletResponse } from "@/lib/api";
import { SUPPORTED_CHAINS } from "./constants";

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
  const [identityLabels, setIdentityLabels] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const loadIdentityLabels = useCallback(async () => {
    try {
      const res = await getWalletIdentityLabels();
      setIdentityLabels(res.identityLabels ?? {});
      return res.identityLabels ?? {};
    } catch {
      setIdentityLabels({});
      return {};
    }
  }, []);

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

  const getPrimaryAddress = useCallback(
    async (chain: string, index: number): Promise<string> => {
      try {
        const res = await getWalletAddress(chain, index);
        return res.address ?? "";
      } catch {
        return "";
      }
    },
    []
  );

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
      if (updates.identityLabel != null && updates.index != null) {
        await patchWalletIdentityLabels({ index: updates.index, label: updates.identityLabel });
      }
      if (updates.defaultChain) {
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
        obj.defaultChain = updates.defaultChain;
        await putConfig(JSON.stringify(obj, null, 2));
      }
    },
    []
  );

  const getIdentityLabel = useCallback(async (index: number): Promise<string> => {
    try {
      const res = await getWalletIdentityLabels();
      return res.identityLabels?.[String(index)] ?? "";
    } catch {
      return "";
    }
  }, []);

  return {
    wallet,
    identityLabels,
    loading,
    error,
    success,
    setError,
    setSuccess,
    loadWallet,
    loadIdentityLabels,
    getPrimaryAddress,
    deriveAddressesForIndex,
    rotateToIndex,
    updateConfigIdentity,
    getIdentityLabel,
  };
}
