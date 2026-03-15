import { useState, useEffect } from "react";
import { AddressCard } from "./AddressCard";
import type { ChainAddress } from "./useWalletConfig";

interface ManageDetailAddressesProps {
  index: number;
  deriveAddressesForIndex: (i: number) => Promise<ChainAddress[]>;
  onCopy: (a: string) => void;
  onShowQr: (addr: string, label: string) => void;
}

export function ManageDetailAddresses({
  index,
  deriveAddressesForIndex,
  onCopy,
  onShowQr,
}: ManageDetailAddressesProps) {
  const [addrs, setAddrs] = useState<ChainAddress[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    deriveAddressesForIndex(index).then(setAddrs).finally(() => setLoading(false));
  }, [index, deriveAddressesForIndex]);

  if (loading) {
    return <div className="col-span-full py-4 text-center text-[#6b8fcc]">Loading…</div>;
  }

  return (
    <>
      {addrs.map((item) => (
        <AddressCard
          key={item.caip2}
          item={item}
          onCopy={onCopy}
          onShowQr={onShowQr}
        />
      ))}
    </>
  );
}
