import { useEffect, useState, useCallback } from "react";
import { getTools, patchToolEnabled, type ToolItem } from "@/lib/api";
import { handleApiError } from "@/lib/api-error";

export function useToolsConfig(hasWriteAccess: boolean) {
  const [tools, setTools] = useState<ToolItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [toggling, setToggling] = useState<Record<string, boolean>>({});

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    getTools()
      .then((res) => setTools(res.tools ?? []))
      .catch((e) => handleApiError(e, setError, "Load failed"))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const handleToggle = useCallback(
    async (tool: ToolItem) => {
      if (!hasWriteAccess) return;
      const nextEnabled = !tool.enabled;
      setToggling((prev) => ({ ...prev, [tool.name]: true }));
      try {
        await patchToolEnabled(tool.name, nextEnabled);
        setTools((prev) =>
          prev.map((t) => (t.name === tool.name ? { ...t, enabled: nextEnabled } : t))
        );
      } catch (e) {
        handleApiError(e, setError, "Update failed");
      } finally {
        setToggling((prev) => ({ ...prev, [tool.name]: false }));
      }
    },
    [hasWriteAccess]
  );

  return { tools, loading, error, toggling, handleToggle };
}
