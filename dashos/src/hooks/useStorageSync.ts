import { useState, useEffect } from "react";

/**
 * Syncs state with a getter, refreshing when a custom event fires.
 * Used by ConfigSubNav and Sidebar to react to visibility preference changes.
 */
export function useStorageSync<T>(
  getValue: () => T,
  changeEvent: string
): T {
  const [value, setValue] = useState(getValue);

  useEffect(() => {
    setValue(getValue());
    const handler = () => setValue(getValue());
    window.addEventListener(changeEvent, handler);
    return () => window.removeEventListener(changeEvent, handler);
  }, [getValue, changeEvent]);

  return value;
}
