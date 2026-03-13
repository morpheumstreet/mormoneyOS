/**
 * Extension Detection Utilities
 * Wallet detection for MetaMask, Phantom, OneKey, OKX, Bitget, etc.
 */

export type ChainType = "ethereum" | "solana" | "bitcoin";

export type WalletName =
  | "OneKey"
  | "OKX"
  | "Bitget"
  | "TokenPocket"
  | "Coinbase Wallet"
  | "Rainbow"
  | "Binance"
  | "MetaMask"
  | "Phantom"
  | "Bitcoin Wallet"
  | "Unisat"
  | null;

function isOneKeyAvailable(): boolean {
  if (typeof window === "undefined") return false;
  const w = window as unknown as { $onekey?: { ethereum?: unknown }; ethereum?: { isOneKey?: boolean } };
  return !!(w.$onekey?.ethereum || w.ethereum?.isOneKey || w.$onekey);
}

function isOKXEthereumAvailable(): boolean {
  if (typeof window === "undefined") return false;
  const w = window as unknown as { okxwallet?: { ethereum?: unknown }; ethereum?: { isOKX?: boolean } };
  return !!(w.okxwallet?.ethereum || w.ethereum?.isOKX);
}

function isBitgetProviderAvailable(): boolean {
  if (typeof window === "undefined") return false;
  const bitkeep = (window as unknown as { bitkeep?: { isBitKeep?: boolean; ethereum?: { isBitEthereum?: boolean } } }).bitkeep;
  if (!bitkeep) return false;
  return bitkeep.isBitKeep === true || bitkeep.ethereum?.isBitEthereum === true;
}

function isTokenPocketProviderAvailable(): boolean {
  if (typeof window === "undefined") return false;
  const w = window as unknown as { tokenpocket?: unknown; ethereum?: { isTokenPocket?: boolean }; tp?: unknown };
  return !!(typeof w.tokenpocket !== "undefined" || w.ethereum?.isTokenPocket === true || (w.tp && typeof w.tp === "object"));
}

function isRainbowAvailable(): boolean {
  if (typeof window === "undefined") return false;
  return (window as unknown as { ethereum?: { isRainbow?: boolean } }).ethereum?.isRainbow === true;
}

function isBinanceProviderAvailable(): boolean {
  if (typeof window === "undefined") return false;
  const w = window as unknown as { binancew3w?: { isExtension?: boolean }; BinanceChain?: unknown; ethereum?: { isBinance?: boolean } };
  return !!(w.binancew3w?.isExtension === true || w.BinanceChain || w.ethereum?.isBinance);
}

function isMetaMaskProviderAvailable(): boolean {
  if (typeof window === "undefined") return false;
  const ethereum = (window as unknown as { ethereum?: { isMetaMask?: boolean; _metamask?: unknown; providerInfo?: { rdns?: string }; isOneKey?: boolean; isOKX?: boolean } }).ethereum;
  if (!ethereum) return false;
  if (ethereum.isMetaMask === true) {
    if (ethereum._metamask && typeof ethereum._metamask === "object") return true;
    if (ethereum.providerInfo?.rdns === "io.metamask") return true;
    const hasOKX = !!(window as unknown as { okxwallet?: unknown }).okxwallet;
    const hasBinance = isBinanceProviderAvailable();
    const hasOneKey = !!(window as unknown as { $onekey?: unknown }).$onekey;
    const hasPhantom = !!(window as unknown as { phantom?: { ethereum?: unknown } }).phantom?.ethereum;
    if (!hasOKX && !hasBinance && !hasOneKey && !hasPhantom) return true;
  }
  return false;
}

function isPhantomAvailable(): boolean {
  if (typeof window === "undefined") return false;
  const w = window as unknown as {
    phantom?: { ethereum?: unknown; solana?: unknown };
    ethereum?: { isPhantom?: boolean; isOneKey?: boolean };
    $onekey?: unknown;
  };
  const hasPhantom = !!(w.phantom?.ethereum || w.phantom?.solana || w.ethereum?.isPhantom);
  return hasPhantom && !w.$onekey && !w.ethereum?.isOneKey;
}

/**
 * Returns the best available wallet for the given chain type.
 * Ethereum preferred for SIWE auth.
 */
export function getBestWallet(chainType: ChainType): WalletName {
  if (chainType === "ethereum") {
    if (isOneKeyAvailable()) return "OneKey";
    if (isOKXEthereumAvailable()) return "OKX";
    if (isBitgetProviderAvailable()) return "Bitget";
    if (isTokenPocketProviderAvailable()) return "TokenPocket";
    if (isRainbowAvailable()) return "Rainbow";
    if (isBinanceProviderAvailable()) return "Binance";
    if (isMetaMaskProviderAvailable()) return "MetaMask";
    if (isPhantomAvailable()) return "Phantom";
  }
  return null;
}
