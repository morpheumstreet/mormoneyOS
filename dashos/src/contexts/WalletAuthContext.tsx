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
  isAuthenticated: boolean;
  connectAndSign: () => Promise<void>;
  bypassDev: () => Promise<void>;
  disconnect: () => void;
}

const WalletAuthContext = createContext<WalletAuthContextValue | null>(null);

const AUTH_API = import.meta.env.VITE_AUTH_API_URL || "";
const API = "/api";

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

async function verifyAndGetTokenConway(
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

/** Dev bypass: POST /api/auth/dev-bypass (when MONEYCLAW_DEV_BYPASS=1) to get token without wallet */
async function bypassAndGetToken(): Promise<{ address: string; token: string }> {
  const res = await fetch(`${API}/auth/dev-bypass`, { method: "POST" });
  if (!res.ok) {
    const err = await res.text();
    throw new Error(err || "Dev bypass failed");
  }
  const data = (await res.json()) as {
    valid?: boolean;
    address?: string;
    token?: string;
    error?: string;
  };
  if (!data.valid || !data.token || !data.address) {
    throw new Error(data.error || "Dev bypass failed");
  }
  return { address: data.address, token: data.token };
}

/** Use mormoneyOS /api/auth/verify: sign message, verify, get JWT for write operations */
async function verifyAndGetTokenLocal(
  address: string,
  message: string,
  signature: string
): Promise<string> {
  const res = await fetch(`${API}/auth/verify`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      chain: "ethereum",
      message,
      signature,
    }),
  });
  if (!res.ok) {
    const err = await res.text();
    throw new Error(err || "Verification failed");
  }
  const data = (await res.json()) as {
    valid?: boolean;
    token?: string;
    error?: string;
  };
  if (!data.valid || !data.token) {
    throw new Error(data.error || "Verification failed");
  }
  return data.token;
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

  const bypassDev = useCallback(async () => {
    setState((s) => ({ ...s, isConnecting: true, error: null }));
    try {
      const { address, token } = await bypassAndGetToken();
      setToken(token);
      setState({
        address,
        walletName: "dev-bypass",
        isConnecting: false,
        isSigning: false,
        error: null,
        hasWriteAccess: true,
      });
    } catch (err) {
      setState((s) => ({
        ...s,
        isConnecting: false,
        error: err instanceof Error ? err.message : "Dev bypass failed",
      }));
      throw err;
    }
  }, []);

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

      setState((s) => ({ ...s, isSigning: true }));

      const { SiweMessage } = await import("siwe");
      const domain =
        typeof window !== "undefined"
          ? window.location.host || "localhost"
          : "localhost";

      let message: string;
      let token: string;

      if (AUTH_API) {
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
        message = siweMessage.prepareMessage();
        const signature = await signEthereumMessage(message, address);
        token = await verifyAndGetTokenConway(AUTH_API, message, signature);
      } else {
        const nonce = crypto.randomUUID?.() ?? `n${Date.now()}`;
        const siweMessage = new SiweMessage({
          domain,
          address,
          statement: "Sign in to MoneyClaw Dashboard for write access.",
          uri: typeof window !== "undefined" ? window.location.origin : "/",
          version: "1",
          chainId: 8453,
          nonce,
        });
        message = siweMessage.prepareMessage();
        const signature = await signEthereumMessage(message, address);
        token = await verifyAndGetTokenLocal(address, message, signature);
      }
      if (!token) {
        throw new Error("No token in response");
      }
      setToken(token);
      setState((s) => ({
        ...s,
        isSigning: false,
        hasWriteAccess: true,
        error: null,
      }));
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
        isAuthenticated: !!state.address && state.hasWriteAccess,
        connectAndSign,
        bypassDev,
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
