/**
 * Generic visibility storage for nav items.
 * Single implementation used by config submenu and sidebar.
 */

export interface VisibilityStorage {
  getHiddenIds(): Set<string>;
  setHiddenIds(ids: Set<string>): void;
}

export function createVisibilityStorage(config: {
  storageKey: string;
  changeEvent: string;
}): VisibilityStorage {
  const { storageKey, changeEvent } = config;

  function getHiddenIds(): Set<string> {
    if (typeof window === "undefined") return new Set();
    try {
      const raw = window.localStorage.getItem(storageKey);
      if (!raw) return new Set();
      const parsed = JSON.parse(raw) as unknown;
      if (!Array.isArray(parsed)) return new Set();
      return new Set(parsed.filter((x): x is string => typeof x === "string"));
    } catch {
      return new Set();
    }
  }

  function setHiddenIds(ids: Set<string>): void {
    if (typeof window === "undefined") return;
    window.localStorage.setItem(storageKey, JSON.stringify([...ids]));
    window.dispatchEvent(new CustomEvent(changeEvent));
  }

  return { getHiddenIds, setHiddenIds };
}
