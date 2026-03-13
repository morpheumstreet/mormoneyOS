import { useState } from "react";
import { getBestWallet } from "@/sdk/utils/extdetection";
import { useWalletAuth } from "@/contexts/WalletAuthContext";

export function WalletConnectScreen() {
  const { connectAndSign, error, isConnecting, isSigning } = useWalletAuth();
  const [submitting, setSubmitting] = useState(false);

  const walletAvailable = !!getBestWallet("ethereum");
  const loading = isConnecting || isSigning || submitting;

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

  return (
    <div className="pairing-shell min-h-screen flex items-center justify-center px-4">
      <div className="pairing-card w-full max-w-md rounded-2xl p-8">
        <div className="text-center mb-6">
          <h1 className="mb-2 text-2xl font-semibold tracking-[0.16em] text-white">
            DashOS
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
          </div>
        ) : (
          <div className="rounded-xl border border-[#2956a8] bg-[#071228]/90 px-4 py-4 text-center">
            <p className="text-sm text-[#a7c4f3]">
              No Ethereum wallet detected. Install MetaMask, Phantom, or another supported wallet extension.
            </p>
          </div>
        )}

        {error && (
          <p className="mt-4 text-center text-sm text-rose-300">{error}</p>
        )}
      </div>
    </div>
  );
}
