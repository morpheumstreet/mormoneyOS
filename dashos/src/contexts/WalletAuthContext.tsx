import {
  createContext,
  useContext,
  useState,
  useCallback,
  useEffect,
  type ReactNode,
} from "react";
import { getBestWallet } from "@/sdk/utils/extdetection";
import { getEthereumAddress, signEthereumMessage } from "@/sdk/signing/ethereum";
import { setToken, clearToken } from "@/lib/auth";

export interface WalletAuthState {
  address: string | null;
  walletName: string | null;
  isConnecting: boolean;
  isSigning: boolean;
  error: string | null;
  hasWriteAccess: boolean;
}

interface WalletAuthContextValue extends WalletAuthState {
  connectAndSign: () => Promise<void>;
  disconnect: () => void;
}

const WalletAuthContext = createContext<WalletAuthContextValue | null>(null);

const AUTH_API = import.meta.env.VITE_AUTH_API_URL || "";

async function fetchNonce(apiBase: string): Promise<string> {
  const res = await fetch(`${apiBase}/v1/auth/nonce`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
  });
  if (!res.ok) throw new Error("Failed to get nonce");
  const data = (await res.json()) as { nonce?: string };
  if (!data.nonce) throw new Error("No nonce in response");
  return data.nonce;
}

async function verifyAndGetToken(
  apiBase: string,
  message: string,
  signature: string
): Promise<string> {
  const res = await fetch(`${apiBase}/v1/auth/verify`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message, signature }),
  });
  if (!res.ok) {
    const err = await res.text();
    throw new Error(err || "Verification failed");
  }
  const data = (await res.json()) as { access_token?: string; token?: string };
  return data.access_token || data.token || "";
}

export function WalletAuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<WalletAuthState>({
    address: null,
    walletName: null,
    isConnecting: false,
    isSigning: false,
    error: null,
    hasWriteAccess: false,
  });

  const disconnect = useCallback(() => {
    clearToken();
    setState({
      address: null,
      walletName: null,
      isConnecting: false,
      isSigning: false,
      error: null,
      hasWriteAccess: false,
    });
  }, []);

  useEffect(() => {
    const handler = () => disconnect();
    window.addEventListener("dashos-unauthorized", handler);
    return () => window.removeEventListener("dashos-unauthorized", handler);
  }, [disconnect]);

  const connectAndSign = useCallback(async () => {
    setState((s) => ({ ...s, isConnecting: true, error: null }));
    try {
      const walletName = getBestWallet("ethereum");
      if (!walletName) {
        throw new Error(
          "No Ethereum wallet detected. Install MetaMask, Phantom, or another supported wallet."
        );
      }

      const address = await getEthereumAddress();
      setState((s) => ({
        ...s,
        address,
        walletName,
        isConnecting: false,
      }));

      if (!AUTH_API) {
        setState((s) => ({
          ...s,
          hasWriteAccess: true,
          error: null,
        }));
        return;
      }

      setState((s) => ({ ...s, isSigning: true }));
      const { SiweMessage } = await import("siwe");
      const nonce = await fetchNonce(AUTH_API);
      const siweMessage = new SiweMessage({
        domain: new URL(AUTH_API).hostname,
        address,
        statement: "Sign in to MoneyClaw Dashboard for write access.",
        uri: `${AUTH_API}/v1/auth/verify`,
        version: "1",
        chainId: 8453,
        nonce,
      });
      const message = siweMessage.prepareMessage();
      const signature = await signEthereumMessage(message, address);
      const token = await verifyAndGetToken(AUTH_API, message, signature);
      if (token) {
        setToken(token);
        setState((s) => ({
          ...s,
          isSigning: false,
          hasWriteAccess: true,
          error: null,
        }));
      } else {
        throw new Error("No token in response");
      }
    } catch (err) {
      setState((s) => ({
        ...s,
        isConnecting: false,
        isSigning: false,
        error: err instanceof Error ? err.message : "Connection failed",
      }));
      throw err;
    }
  }, []);

  return (
    <WalletAuthContext.Provider
      value={{
        ...state,
        connectAndSign,
        disconnect,
      }}
    >
      {children}
    </WalletAuthContext.Provider>
  );
}

export function useWalletAuth() {
  const ctx = useContext(WalletAuthContext);
  if (!ctx) throw new Error("useWalletAuth must be used within WalletAuthProvider");
  return ctx;
}
