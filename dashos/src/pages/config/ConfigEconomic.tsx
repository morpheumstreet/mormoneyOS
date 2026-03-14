import { useEffect, useState } from "react";
import {
  Wallet,
  AlertTriangle,
  Loader2,
  Save,
  CheckCircle,
  DollarSign,
} from "lucide-react";
import { useWalletAuth } from "@/contexts/WalletAuthContext";
import {
  getEconomic,
  putEconomic,
  type EconomicResponse,
  type TreasuryPolicy,
} from "@/lib/api";

const CHAIN_LABELS: Record<string, string> = {
  "eip155:8453": "Base",
  "eip155:84532": "Base Sepolia",
  "eip155:1": "Ethereum",
  "eip155:137": "Polygon",
  "eip155:42161": "Arbitrum",
};

export default function ConfigEconomic() {
  const { hasWriteAccess } = useWalletAuth();
  const [data, setData] = useState<EconomicResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [resourceMode, setResourceMode] = useState<
    "auto" | "forced_on" | "forced_off"
  >("auto");
  const [treasury, setTreasury] = useState<TreasuryPolicy>({});

  useEffect(() => {
    getEconomic()
      .then((res) => {
        setData(res);
        setResourceMode(res.resourceConstraintMode || "auto");
        setTreasury(res.treasuryPolicy || {});
      })
      .catch((e) => setError(e instanceof Error ? e.message : "Load failed"))
      .finally(() => setLoading(false));
  }, []);

  const handleSave = async () => {
    if (!hasWriteAccess) {
      setError("Write access required. Connect wallet and sign.");
      return;
    }
    setSaving(true);
    setError(null);
    setSuccess(null);
    try {
      await putEconomic({
        resourceConstraintMode: resourceMode,
        treasuryPolicy: treasury,
      });
      setSuccess("Economic settings saved.");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="electric-loader h-12 w-12 rounded-full" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="electric-icon h-10 w-10 rounded-xl flex items-center justify-center">
            <Wallet className="h-5 w-5 text-[#9bc3ff]" />
          </div>
          <div>
            <h2 className="text-lg font-semibold text-white">Economic</h2>
            <p className="text-sm text-[#8aa8df]">
              Wallet addresses, USDC balances, and economic constraints
            </p>
          </div>
        </div>
        <button
          onClick={handleSave}
          disabled={saving || !hasWriteAccess}
          className="electric-button flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50"
        >
          <Save className="h-4 w-4" />
          {saving ? "Saving…" : "Save"}
        </button>
      </div>

      {!hasWriteAccess && (
        <div className="electric-card p-4 border-amber-500/30 bg-amber-950/20">
          <div className="flex items-start gap-3">
            <AlertTriangle className="h-5 w-5 text-amber-400 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-amber-200">
                Write access required
              </p>
              <p className="text-sm text-amber-300/80 mt-1">
                Connect your wallet and sign to edit economic constraints.
              </p>
            </div>
          </div>
        </div>
      )}

      {success && (
        <div className="electric-card p-3 border-emerald-500/30 bg-emerald-950/20 flex items-center gap-2">
          <CheckCircle className="h-4 w-4 text-emerald-400 flex-shrink-0" />
          <span className="text-sm text-emerald-300">{success}</span>
        </div>
      )}

      {error && (
        <div className="electric-card p-3 border-rose-500/30 bg-rose-950/20 flex items-center gap-2">
          <AlertTriangle className="h-4 w-4 text-rose-400 flex-shrink-0" />
          <span className="text-sm text-rose-300">{error}</span>
        </div>
      )}

      {/* Wallet addresses and USDC balances */}
      <div className="electric-card p-6 space-y-4">
        <h3 className="text-sm font-medium text-white flex items-center gap-2">
          <DollarSign className="h-4 w-4" />
          Wallet addresses & USDC balances
        </h3>
        {data?.balances && data.balances.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-[#1a3670] text-left text-[#8aa8df]">
                  <th className="pb-2 pr-4">Address</th>
                  <th className="pb-2 pr-4">Chain</th>
                  <th className="pb-2 pr-4">Source</th>
                  <th className="pb-2 text-right">USDC</th>
                </tr>
              </thead>
              <tbody>
                {data.balances.map((b, i) => (
                  <tr
                    key={i}
                    className="border-b border-[#1a3670]/50 text-white"
                  >
                    <td className="py-2 pr-4 font-mono text-xs">
                      {b.address}
                    </td>
                    <td className="py-2 pr-4">
                      {CHAIN_LABELS[b.chain] || b.chain}
                    </td>
                    <td className="py-2 pr-4 text-[#8aa8df]">{b.source}</td>
                    <td className="py-2 text-right">
                      {b.error ? (
                        <span className="text-rose-400 text-xs">{b.error}</span>
                      ) : b.balance != null ? (
                        <span className="text-emerald-300">
                          ${b.balance.toFixed(2)}
                        </span>
                      ) : (
                        "—"
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <p className="text-sm text-[#6b8fcc]">
            No wallet addresses found. Configure walletAddress or creatorAddress
            in General config.
          </p>
        )}
      </div>

      {/* Economic constraint mode */}
      <div className="electric-card p-6 space-y-4">
        <h3 className="text-sm font-medium text-white">
          Economic constraint mode
        </h3>
        <p className="text-xs text-[#6b8fcc]">
          Controls when inference uses the cheaper low-compute model based on
          credits and survival tier.
        </p>
        <div className="flex items-center gap-2">
          <select
            value={resourceMode}
            onChange={(e) =>
              setResourceMode(
                e.target.value as "auto" | "forced_on" | "forced_off"
              )
            }
            disabled={!hasWriteAccess}
            className="rounded-lg border border-[#29509c] bg-[#071228]/90 px-3 py-2 text-sm text-white focus:border-[#4f83ff] focus:outline-none disabled:opacity-60"
          >
            <option value="auto">Auto (credit-based)</option>
            <option value="forced_on">Force on (always low compute)</option>
            <option value="forced_off">Force off (always full compute)</option>
          </select>
        </div>
      </div>

      {/* Treasury policy */}
      <div className="electric-card p-6 space-y-4">
        <h3 className="text-sm font-medium text-white">Treasury policy</h3>
        <p className="text-xs text-[#6b8fcc]">
          Financial limits for transfers and inference spend. Values in cents.
        </p>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="mb-1 block text-xs font-medium text-[#8aa8df]">
              Max single transfer (¢)
            </label>
            <input
              type="number"
              min={0}
              value={treasury.maxSingleTransferCents ?? 5000}
              onChange={(e) =>
                setTreasury((t) => ({
                  ...t,
                  maxSingleTransferCents: parseInt(e.target.value, 10) || 0,
                }))
              }
              disabled={!hasWriteAccess}
              className="w-full rounded-lg border border-[#29509c] bg-[#071228]/90 px-3 py-2 text-sm text-white focus:border-[#4f83ff] focus:outline-none disabled:opacity-60"
            />
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-[#8aa8df]">
              Max hourly transfer (¢)
            </label>
            <input
              type="number"
              min={0}
              value={treasury.maxHourlyTransferCents ?? 10000}
              onChange={(e) =>
                setTreasury((t) => ({
                  ...t,
                  maxHourlyTransferCents: parseInt(e.target.value, 10) || 0,
                }))
              }
              disabled={!hasWriteAccess}
              className="w-full rounded-lg border border-[#29509c] bg-[#071228]/90 px-3 py-2 text-sm text-white focus:border-[#4f83ff] focus:outline-none disabled:opacity-60"
            />
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-[#8aa8df]">
              Max daily transfer (¢)
            </label>
            <input
              type="number"
              min={0}
              value={treasury.maxDailyTransferCents ?? 50000}
              onChange={(e) =>
                setTreasury((t) => ({
                  ...t,
                  maxDailyTransferCents: parseInt(e.target.value, 10) || 0,
                }))
              }
              disabled={!hasWriteAccess}
              className="w-full rounded-lg border border-[#29509c] bg-[#071228]/90 px-3 py-2 text-sm text-white focus:border-[#4f83ff] focus:outline-none disabled:opacity-60"
            />
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-[#8aa8df]">
              Min reserve (¢)
            </label>
            <input
              type="number"
              min={0}
              value={treasury.minReserveCents ?? 100}
              onChange={(e) =>
                setTreasury((t) => ({
                  ...t,
                  minReserveCents: parseInt(e.target.value, 10) || 0,
                }))
              }
              disabled={!hasWriteAccess}
              className="w-full rounded-lg border border-[#29509c] bg-[#071228]/90 px-3 py-2 text-sm text-white focus:border-[#4f83ff] focus:outline-none disabled:opacity-60"
            />
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-[#8aa8df]">
              Inference daily budget (¢)
            </label>
            <input
              type="number"
              min={0}
              value={treasury.inferenceDailyBudgetCents ?? 5000}
              onChange={(e) =>
                setTreasury((t) => ({
                  ...t,
                  inferenceDailyBudgetCents: parseInt(e.target.value, 10) || 0,
                }))
              }
              disabled={!hasWriteAccess}
              className="w-full rounded-lg border border-[#29509c] bg-[#071228]/90 px-3 py-2 text-sm text-white focus:border-[#4f83ff] focus:outline-none disabled:opacity-60"
            />
          </div>
        </div>
      </div>
    </div>
  );
}
