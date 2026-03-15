import { useEffect, useState, useCallback } from "react";
import {
  getSoulConfig,
  putSoulConfig,
  postSoulEnhance,
  type SoulConfig,
} from "@/lib/api";
import { getApiErrorMessage } from "@/lib/api-error";

const MIN_ENHANCE_WORDS = 5;

export function useSoulConfig(hasWriteAccess: boolean) {
  const [config, setConfig] = useState<SoulConfig>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [enhanceWords, setEnhanceWords] = useState("");
  const [enhancing, setEnhancing] = useState(false);

  useEffect(() => {
    getSoulConfig()
      .then(setConfig)
      .catch((e) => setError(getApiErrorMessage(e, "Load failed")))
      .finally(() => setLoading(false));
  }, []);

  const clearFeedback = useCallback(() => {
    setError(null);
    setSuccess(null);
  }, []);

  const handleSave = useCallback(async () => {
    if (!hasWriteAccess) {
      setError("Write access required. Connect wallet and sign.");
      return;
    }
    setSaving(true);
    clearFeedback();
    try {
      await putSoulConfig(config);
      setSuccess("Soul config saved.");
    } catch (e) {
      setError(getApiErrorMessage(e, "Save failed"));
    } finally {
      setSaving(false);
    }
  }, [config, hasWriteAccess, clearFeedback]);

  const updateConstraint = useCallback((idx: number, value: string) => {
    setConfig((c) => {
      const arr = [...(c.behavioralConstraints || [])];
      arr[idx] = value;
      return { ...c, behavioralConstraints: arr };
    });
  }, []);

  const addConstraint = useCallback(() => {
    setConfig((c) => ({
      ...c,
      behavioralConstraints: [...(c.behavioralConstraints || []), ""],
    }));
  }, []);

  const removeConstraint = useCallback((idx: number) => {
    setConfig((c) => {
      const arr = [...(c.behavioralConstraints || [])];
      arr.splice(idx, 1);
      return { ...c, behavioralConstraints: arr };
    });
  }, []);

  const handleEnhance = useCallback(async (apply: boolean) => {
    const words = enhanceWords.trim();
    if (!words) {
      setError("Enter a few words to enhance.");
      return;
    }
    const wordCount = words.split(/\s+/).filter(Boolean).length;
    if (wordCount < MIN_ENHANCE_WORDS) {
      setError(`Enter at least ${MIN_ENHANCE_WORDS} words to enhance.`);
      return;
    }
    if (!hasWriteAccess && apply) {
      setError("Write access required to apply. Connect wallet and sign.");
      return;
    }
    setEnhancing(true);
    clearFeedback();
    try {
      const res = await postSoulEnhance(words, apply);
      setConfig((c) => ({ ...c, systemPrompt: res.systemPrompt }));
      if (apply) {
        setSuccess("System prompt enhanced and saved.");
        setEnhanceWords("");
      } else {
        setSuccess("Preview ready. Click 'Enhance & Apply' to save.");
      }
    } catch (e) {
      setError(getApiErrorMessage(e, "Enhance failed"));
    } finally {
      setEnhancing(false);
    }
  }, [enhanceWords, hasWriteAccess, clearFeedback]);

  const updateConfig = useCallback(<K extends keyof SoulConfig>(
    key: K,
    value: SoulConfig[K]
  ) => {
    setConfig((c) => ({ ...c, [key]: value }));
  }, []);

  return {
    config,
    loading,
    saving,
    error,
    success,
    enhanceWords,
    setEnhanceWords,
    enhancing,
    handleSave,
    handleEnhance,
    updateConfig,
    updateConstraint,
    addConstraint,
    removeConstraint,
  };
}
