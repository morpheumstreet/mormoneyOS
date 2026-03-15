import { createContext, useContext, type ReactNode } from "react";
import { useWalletConfig } from "./useWalletConfig";
import type { ChainAddress } from "./useWalletConfig";

type ConfigWalletContextValue = ReturnType<typeof useWalletConfig>;

const ConfigWalletContext = createContext<ConfigWalletContextValue | null>(null);

export function ConfigWalletProvider({ children }: { children: ReactNode }) {
  const value = useWalletConfig();
  return (
    <ConfigWalletContext.Provider value={value}>
      {children}
    </ConfigWalletContext.Provider>
  );
}

export function useConfigWallet(): ConfigWalletContextValue {
  const ctx = useContext(ConfigWalletContext);
  if (!ctx) throw new Error("useConfigWallet must be used within ConfigWalletProvider");
  return ctx;
}

export type { ChainAddress };
