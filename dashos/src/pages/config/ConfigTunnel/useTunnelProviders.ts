import { useEffect, useState, useCallback } from "react";
import {
  getTunnelProviders,
  getTunnels,
  putTunnelProvider,
  postTunnelProviderRestart,
  type TunnelProviderField,
  type TunnelProviderSchema,
} from "@/lib/api";
import { MASKED_PLACEHOLDER } from "./constants";

export interface TunnelProvidersData {
  providers: string[];
  schemas: Record<string, TunnelProviderSchema>;
  config: { defaultProvider: string; providers: Record<string, Record<string, unknown>> };
}

export interface TunnelItem {
  port: number;
  provider: string;
  public_url: string;
}

export function useTunnelProviders(hasWriteAccess: boolean) {
  const [providersData, setProvidersData] = useState<TunnelProvidersData | null>(null);
  const [tunnels, setTunnels] = useState<TunnelItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  const [formValues, setFormValues] = useState<
    Record<string, Record<string, string | boolean>>
  >({});
  const [saving, setSaving] = useState<Record<string, boolean>>({});
  const [restarting, setRestarting] = useState<Record<string, boolean>>({});
  const [toggling, setToggling] = useState<Record<string, boolean>>({});

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [provRes, tunnelsRes] = await Promise.all([
        getTunnelProviders(),
        getTunnels(),
      ]);
      setProvidersData(provRes);
      setTunnels(tunnelsRes.tunnels || []);

      const initial: Record<string, Record<string, string | boolean>> = {};
      const provConfig = provRes.config?.providers || {};
      const schemas = provRes.schemas || {};
      for (const name of provRes.providers || []) {
        const pc = provConfig[name] as Record<string, unknown> | undefined;
        const schema = schemas[name];
        if (schema?.fields) {
          const vals: Record<string, string | boolean> = {};
          for (const f of schema.fields) {
            const v = pc?.[f.name];
            if (f.type === "boolean") {
              vals[f.name] = !!v;
            } else if (f.type === "password") {
              vals[f.name] = "";
            } else {
              vals[f.name] = (typeof v === "string" ? v : "") || "";
            }
          }
          initial[name] = vals;
        }
      }
      setFormValues(initial);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Load failed");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleSaveConfig = useCallback(
    async (name: string) => {
      if (!hasWriteAccess || !providersData) return;
      const schema = providersData.schemas[name];
      if (!schema?.fields?.length) return;

      setSaving((prev) => ({ ...prev, [name]: true }));
      setError(null);
      try {
        const vals = formValues[name] || {};
        const body: Record<string, unknown> = {};
        const pc = providersData.config?.providers?.[name] as Record<string, unknown> | undefined;
        body.enabled = !!pc?.enabled;
        for (const f of schema.fields) {
          const v = vals[f.name];
          if (f.type === "boolean") {
            body[f.name] = !!v;
          } else if (f.type === "password") {
            if (typeof v === "string" && v !== "" && v !== MASKED_PLACEHOLDER) {
              body[f.name] = v;
            }
          } else {
            if (typeof v === "string" && v !== "") {
              body[f.name] = v;
            }
          }
        }
        await putTunnelProvider(name, body);
        await fetchData();
        setExpanded((prev) => ({ ...prev, [name]: false }));
      } catch (e) {
        setError(e instanceof Error ? e.message : "Save failed");
      } finally {
        setSaving((prev) => ({ ...prev, [name]: false }));
      }
    },
    [hasWriteAccess, providersData, formValues, fetchData]
  );

  const handleToggleEnabled = useCallback(
    async (name: string) => {
      if (!hasWriteAccess || !providersData) return;
      const pc = providersData.config?.providers?.[name] as Record<string, unknown> | undefined;
      const nextEnabled = !pc?.enabled;
      setToggling((prev) => ({ ...prev, [name]: true }));
      setError(null);
      try {
        await putTunnelProvider(name, { enabled: nextEnabled });
        await fetchData();
      } catch (e) {
        setError(e instanceof Error ? e.message : "Update failed");
      } finally {
        setToggling((prev) => ({ ...prev, [name]: false }));
      }
    },
    [hasWriteAccess, providersData, fetchData]
  );

  const handleRestart = useCallback(
    async (name: string) => {
      if (!hasWriteAccess) return;
      setRestarting((prev) => ({ ...prev, [name]: true }));
      setError(null);
      try {
        await postTunnelProviderRestart(name);
        await fetchData();
      } catch (e) {
        setError(e instanceof Error ? e.message : "Restart failed");
      } finally {
        setRestarting((prev) => ({ ...prev, [name]: false }));
      }
    },
    [hasWriteAccess, fetchData]
  );

  const updateFormValue = useCallback(
    (providerName: string, key: string, value: string | boolean) => {
      setFormValues((prev) => ({
        ...prev,
        [providerName]: {
          ...(prev[providerName] || {}),
          [key]: value,
        },
      }));
    },
    []
  );

  const isConfigured = useCallback(
    (name: string, field: TunnelProviderField) => {
      if (field.type !== "password") return false;
      const pc = providersData?.config?.providers?.[name] as Record<string, unknown> | undefined;
      const v = pc?.[field.name];
      return typeof v === "string" && v !== "";
    },
    [providersData]
  );

  const setExpandedByName = useCallback((name: string) => {
    setExpanded((prev) => ({ ...prev, [name]: !prev[name] }));
  }, []);

  return {
    providersData,
    tunnels,
    loading,
    error,
    expanded,
    formValues,
    saving,
    restarting,
    toggling,
    fetchData,
    handleSaveConfig,
    handleToggleEnabled,
    handleRestart,
    updateFormValue,
    isConfigured,
    setExpandedByName,
  };
}
