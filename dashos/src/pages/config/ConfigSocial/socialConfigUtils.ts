/** Config value parsing — single source of truth for form ↔ config conversion */

import type { SocialConfigField } from "@/lib/api";
import { MASKED_PLACEHOLDER } from "./constants";

export function formatArrayValue(val: unknown): string {
  if (Array.isArray(val)) return val.join(", ");
  if (typeof val === "string") return val;
  return "";
}

/** Build form values from channel config (for initial load). */
export function configToFormValues(
  configFields: SocialConfigField[],
  config: Record<string, unknown> | undefined
): Record<string, string | boolean> {
  const vals: Record<string, string | boolean> = {};
  for (const f of configFields) {
    const v = config?.[f.key];
    if (f.type === "boolean") {
      vals[f.key] = !!v;
    } else if (f.type === "array") {
      vals[f.key] = formatArrayValue(v);
    } else if (f.type === "password") {
      vals[f.key] = ""; // Never prefill password
    } else {
      vals[f.key] = (typeof v === "string" ? v : "") || "";
    }
  }
  return vals;
}

/** Build API config payload from form values. */
export function formValuesToConfig(
  configFields: SocialConfigField[],
  vals: Record<string, string | boolean>
): Record<string, unknown> {
  const config: Record<string, unknown> = {};
  for (const f of configFields) {
    const v = vals[f.key];
    if (f.type === "boolean") {
      config[f.key] = !!v;
    } else if (f.type === "array") {
      const str = typeof v === "string" ? v : "";
      config[f.key] = str
        ? str.split(",").map((s) => s.trim()).filter(Boolean)
        : [];
    } else if (f.type === "password") {
      if (typeof v === "string" && v !== "" && v !== MASKED_PLACEHOLDER) {
        config[f.key] = v;
      }
    } else {
      if (typeof v === "string" && v !== "") {
        config[f.key] = v;
      }
    }
  }
  return config;
}
