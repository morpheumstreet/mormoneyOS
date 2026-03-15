/** Supported chains for identity derivation (CAIP-2). Morpheum highlighted per design. */
export const SUPPORTED_CHAINS = [
  { caip2: "eip155:8453", name: "Base", isMorpheum: false },
  { caip2: "eip155:10900", name: "Morpheum", isMorpheum: true },
  { caip2: "eip155:1", name: "Ethereum", isMorpheum: false },
  { caip2: "eip155:56", name: "BNB Chain", isMorpheum: false },
  { caip2: "bip122:000000000019d6689c085ae165831e93", name: "Bitcoin", isMorpheum: false },
  { caip2: "tron:728126428", name: "Tron", isMorpheum: false },
  { caip2: "sui:mainnet", name: "Sui", isMorpheum: false },
  { caip2: "polkadot:91b171bb158e2d3848fa23a9f1c25182", name: "Polkadot", isMorpheum: false },
  { caip2: "xrpl:0", name: "XRP Ledger", isMorpheum: false },
] as const;

export const DEFAULT_CHAIN = "eip155:8453";
