import { useState, useEffect } from "react";
import { getBestWallet } from "@/sdk/utils/extdetection";
import { useWalletAuth } from "@/contexts/WalletAuthContext";
import { getAuthConfigGuestEnabled } from "@/lib/api";

const isDev = import.meta.env.DEV;

export function WalletConnectScreen() {
  const { connectAndSign, connectAsGuest, bypassDev, error, isConnecting, isSigning } =
    useWalletAuth();
  const [submitting, setSubmitting] = useState(false);
  const [guestEnabled, setGuestEnabled] = useState(false);

  useEffect(() => {
    getAuthConfigGuestEnabled()
      .then((r) => setGuestEnabled(r.guest_access_enabled))
      .catch(() => setGuestEnabled(false));
  }, []);

  const walletAvailable = !!getBestWallet("ethereum");
  const loading = isConnecting || isSigning || submitting;

  const handleBypass = async () => {
    setSubmitting(true);
    try {
      await bypassDev();
    } catch {
      // Error shown via context
    } finally {
      setSubmitting(false);
    }
  };

  const handleConnect = async () => {
    setSubmitting(true);
    try {
      await connectAndSign();
    } catch {
      // Error shown via context
    } finally {
      setSubmitting(false);
    }
  };

  const handleGuestLogin = () => {
    connectAsGuest();
  };

  return (
    <div className="pairing-shell min-h-screen flex items-center justify-center px-4">
      <div className="pairing-card w-full max-w-md rounded-2xl p-8">
        <div className="text-center mb-6">
          <h1 className="mb-2 text-2xl font-semibold tracking-[0.16em] text-white">
          MormOS
          </h1>
          <p className="text-sm text-[#9bb8e8]">
            Connect your wallet and sign to authorize write access to MoneyClaw Command Center.
          </p>
        </div>

        {walletAvailable ? (
          <div className="space-y-4">
            <p className="text-xs uppercase tracking-[0.13em] text-[#7ea5eb]">
              Use your browser wallet extension (MetaMask, Phantom, etc.)
            </p>
            <button
              type="button"
              onClick={handleConnect}
              disabled={loading}
              className="electric-button w-full rounded-xl py-3 font-medium text-white disabled:opacity-50"
            >
              {loading
                ? isSigning
                  ? "Sign message in wallet..."
                  : "Connecting..."
                : "Connect Wallet & Sign"}
            </button>
            {guestEnabled && (
              <button
                type="button"
                onClick={handleGuestLogin}
                disabled={loading}
                className="w-full rounded-xl border border-[#2956a8] bg-[#071228]/60 py-2.5 text-sm font-medium text-[#9bb8e8] hover:bg-[#071228] disabled:opacity-50"
              >
                Continue as Guest (read-only)
              </button>
            )}
            {isDev && (
              <button
                type="button"
                onClick={handleBypass}
                disabled={loading}
                className="w-full rounded-xl border border-amber-500/60 bg-amber-500/10 py-2.5 text-sm font-medium text-amber-300 hover:bg-amber-500/20 disabled:opacity-50"
              >
                Dev bypass (no wallet)
              </button>
            )}
          </div>
        ) : (
          <div className="space-y-4">
            <div className="rounded-xl border border-[#2956a8] bg-[#071228]/90 px-4 py-4 text-center">
              <p className="text-sm text-[#a7c4f3]">
                No Ethereum wallet detected. Install MetaMask, Phantom, or another supported wallet extension.
              </p>
            </div>
            {guestEnabled && (
              <button
                type="button"
                onClick={handleGuestLogin}
                disabled={loading}
                className="w-full rounded-xl border border-[#2956a8] bg-[#071228]/60 py-2.5 text-sm font-medium text-[#9bb8e8] hover:bg-[#071228] disabled:opacity-50"
              >
                Continue as Guest (read-only)
              </button>
            )}
            {isDev && (
              <button
                type="button"
                onClick={handleBypass}
                disabled={loading}
                className="w-full rounded-xl border border-amber-500/60 bg-amber-500/10 py-2.5 text-sm font-medium text-amber-300 hover:bg-amber-500/20 disabled:opacity-50"
              >
                Dev bypass (no wallet)
              </button>
            )}
          </div>
        )}

        {error && (
          <p className="mt-4 text-center text-sm text-rose-300">{error}</p>
        )}
      </div>
    </div>
  );
}
