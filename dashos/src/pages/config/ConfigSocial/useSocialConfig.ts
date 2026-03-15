import { useEffect, useState, useCallback } from "react";
import {
  getSocial,
  patchSocialEnabled,
  putSocialConfig,
  type SocialChannelItem,
} from "@/lib/api";
import { getApiErrorMessage } from "@/lib/api-error";
import { ERROR_MESSAGES } from "./constants";
import {
  configToFormValues,
  formValuesToConfig,
} from "./socialConfigUtils";

export function useSocialConfig(hasWriteAccess: boolean) {
  const [channels, setChannels] = useState<SocialChannelItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [toggling, setToggling] = useState<Record<string, boolean>>({});
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  const [formValues, setFormValues] = useState<
    Record<string, Record<string, string | boolean>>
  >({});
  const [saving, setSaving] = useState<Record<string, boolean>>({});

  const fetchChannels = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await getSocial();
      setChannels(res.channels || []);
      const initial: Record<string, Record<string, string | boolean>> = {};
      for (const c of res.channels || []) {
        if (c.configFields && c.config) {
          initial[c.name] = configToFormValues(c.configFields, c.config);
        }
      }
      setFormValues(initial);
    } catch (e) {
      setError(getApiErrorMessage(e, ERROR_MESSAGES.LOAD_FAILED));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchChannels();
  }, [fetchChannels]);

  const handleToggle = useCallback(
    async (channel: SocialChannelItem) => {
      if (!hasWriteAccess) return;
      const nextEnabled = !channel.enabled;
      setToggling((prev) => ({ ...prev, [channel.name]: true }));
      try {
        await patchSocialEnabled(channel.name, nextEnabled);
        setChannels((prev) =>
          prev.map((c) =>
            c.name === channel.name ? { ...c, enabled: nextEnabled } : c
          )
        );
      } catch (e) {
        setError(getApiErrorMessage(e, ERROR_MESSAGES.UPDATE_FAILED));
      } finally {
        setToggling((prev) => ({ ...prev, [channel.name]: false }));
      }
    },
    [hasWriteAccess]
  );

  const handleSaveConfig = useCallback(
    async (channel: SocialChannelItem) => {
      if (!hasWriteAccess || !channel.configFields) return;
      setSaving((prev) => ({ ...prev, [channel.name]: true }));
      setError(null);
      try {
        const vals = formValues[channel.name] || {};
        const config = formValuesToConfig(channel.configFields, vals);
        const res = await putSocialConfig(channel.name, config);
        if (res.ok && res.validated) {
          const fresh = await getSocial();
          setChannels(fresh.channels || []);
          setExpanded((prev) => ({ ...prev, [channel.name]: false }));
        } else {
          setError(res.error || ERROR_MESSAGES.VALIDATION_FAILED);
        }
      } catch (e) {
        setError(getApiErrorMessage(e, ERROR_MESSAGES.SAVE_FAILED));
      } finally {
        setSaving((prev) => ({ ...prev, [channel.name]: false }));
      }
    },
    [hasWriteAccess, formValues]
  );

  const updateFormValue = useCallback(
    (channelName: string, key: string, value: string | boolean) => {
      setFormValues((prev) => ({
        ...prev,
        [channelName]: {
          ...(prev[channelName] || {}),
          [key]: value,
        },
      }));
    },
    []
  );

  const toggleExpanded = useCallback((channelName: string) => {
    setExpanded((prev) => ({ ...prev, [channelName]: !prev[channelName] }));
  }, []);

  return {
    channels,
    loading,
    error,
    toggling,
    expanded,
    formValues,
    saving,
    handleToggle,
    handleSaveConfig,
    updateFormValue,
    toggleExpanded,
  };
}
