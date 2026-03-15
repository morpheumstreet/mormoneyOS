import { DollarSign } from "lucide-react";
import type { EconomicBalance } from "@/lib/api";
import { CHAIN_LABELS } from "./constants";

interface BalancesTableProps {
  balances: EconomicBalance[];
}

const EMPTY_MESSAGE =
  "No wallet addresses found. Configure walletAddress or creatorAddress in General config.";

export function BalancesTable({ balances }: BalancesTableProps) {
  const hasBalances = balances && balances.length > 0;

  return (
    <div className="electric-card p-6 space-y-4">
      <h3 className="text-sm font-medium text-white flex items-center gap-2">
        <DollarSign className="h-4 w-4" />
        Wallet addresses & USDC balances
      </h3>
      {hasBalances ? (
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
              {balances.map((b, i) => (
                <tr
                  key={`${b.address}-${b.chain}-${i}`}
                  className="border-b border-[#1a3670]/50 text-white"
                >
                  <td className="py-2 pr-4 font-mono text-xs">{b.address}</td>
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
        <p className="text-sm text-[#6b8fcc]">{EMPTY_MESSAGE}</p>
      )}
    </div>
  );
}
