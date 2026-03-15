/** Shared theme class names — DRY replacement for repeated Tailwind strings */

export const inputBase =
  "rounded border border-[#29509c] bg-[#071228]/90 text-sm text-white placeholder:text-[#6b8fcc] focus:border-[#4f83ff] focus:outline-none disabled:opacity-60";

export const inputSm = `${inputBase} px-2 py-1.5`;
export const inputMd = `${inputBase} px-3 py-2 rounded-lg`;

/** Full-width input/textarea styling for config forms. */
export const inputConfig = `w-full ${inputMd}`;

export const filterButtonActive =
  "bg-[#4f83ff]/30 text-[#9bc3ff] border border-[#4f83ff]/50";
export const filterButtonInactive =
  "bg-[#071228]/80 text-[#8aa8df] border border-[#29509c] hover:bg-[#07132f]";

export function filterButtonClass(active: boolean): string {
  return `px-2.5 py-1 rounded text-xs font-medium transition-colors ${
    active ? filterButtonActive : filterButtonInactive
  }`;
}

export function filterButtonClassLg(active: boolean): string {
  return `flex items-center gap-1.5 px-3 py-1.5 rounded text-sm font-medium transition-colors ${
    active ? filterButtonActive : filterButtonInactive
  }`;
}
