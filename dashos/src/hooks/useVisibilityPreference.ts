import { useState, useCallback } from "react";
import type { VisibilityStorage } from "@/lib/storageVisibility";

export function useVisibilityPreference(storage: VisibilityStorage) {
  const [hiddenIds, setHiddenIds] = useState<Set<string>>(storage.getHiddenIds);

  const toggle = useCallback(
    (id: string) => {
      setHiddenIds((prev) => {
        const next = new Set(prev);
        if (next.has(id)) next.delete(id);
        else next.add(id);
        storage.setHiddenIds(next);
        return next;
      });
    },
    [storage]
  );

  return [hiddenIds, toggle] as const;
}
