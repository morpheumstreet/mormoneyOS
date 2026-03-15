import { useState, useCallback } from "react";
import { getConfig, putConfig, putProviderEndpoint } from "@/lib/api";
import { handleApiError } from "@/lib/api-error";
import type { ModelProvider } from "@/lib/api";
import type { ModelsResponse } from "@/lib/api";

export function useProviderKeys(providers: ModelProvider[]) {
  const [apiKeysOpen, setApiKeysOpen] = useState(false);
  const [providerKeyValues, setProviderKeyValues] = useState<
    Record<string, string>
  >({});
  const [savingProviderKey, setSavingProviderKey] = useState<string | null>(
    null
  );

  const mergeEndpointValuesFromResponse = useCallback((res: ModelsResponse) => {
    const epValues: Record<string, string> = {};
    for (const p of res.providers || []) {
      if (p.endpointConfigKey && p.endpointValue) {
        epValues[p.endpointConfigKey] = p.endpointValue;
      }
    }
    if (Object.keys(epValues).length) {
      setProviderKeyValues((prev) => ({ ...prev, ...epValues }));
    }
  }, []);

  const saveProviderKey = useCallback(
    async (
      configKey: string,
      value: string,
      hasWriteAccess: boolean,
      setError: (s: string | null) => void,
      load: () => void
    ) => {
      if (!hasWriteAccess || !configKey) return;
      setSavingProviderKey(configKey);
      setError(null);
      try {
        const raw = await getConfig();
        const parsed = JSON.parse(raw || "{}") as Record<string, unknown>;
        parsed[configKey] = value || undefined;
        await putConfig(JSON.stringify(parsed));
        setProviderKeyValues((prev) => ({ ...prev, [configKey]: "" }));
        load();
      } catch (e) {
        handleApiError(e, setError, "Failed to save API key");
      } finally {
        setSavingProviderKey(null);
      }
    },
    []
  );

  const saveProviderEndpoint = useCallback(
    async (
      providerKey: string,
      url: string,
      hasWriteAccess: boolean,
      setError: (s: string | null) => void,
      load: () => void
    ) => {
      if (!hasWriteAccess || !providerKey) return;
      const configKey = providers.find(
        (p) => p.key === providerKey
      )?.endpointConfigKey;
      if (!configKey) return;
      setSavingProviderKey(configKey);
      setError(null);
      try {
        await putProviderEndpoint(providerKey, url.trim());
        load();
      } catch (e) {
        handleApiError(e, setError, "Failed to save endpoint URL");
      } finally {
        setSavingProviderKey(null);
      }
    },
    [providers]
  );

  return {
    apiKeysOpen,
    setApiKeysOpen,
    providerKeyValues,
    setProviderKeyValues,
    savingProviderKey,
    mergeEndpointValuesFromResponse,
    saveProviderKey,
    saveProviderEndpoint,
  };
}
