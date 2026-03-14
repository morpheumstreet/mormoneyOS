/**
 * Ethereum signing - personal_sign (eth_sign) for SIWE and message signing.
 */

export interface EthereumProvider {
  request(args: { method: "eth_requestAccounts" }): Promise<string[]>;
  request(args: { method: "personal_sign"; params: [string, string] }): Promise<string>;
  request(args: { method: "eth_sign"; params: [string, string] }): Promise<string>;
  request(args: { method: "eth_chainId" }): Promise<string>;
}

function getEthereumProvider(): EthereumProvider | null {
  if (typeof window === "undefined") return null;
  const w = window as unknown as {
    $onekey?: { ethereum?: EthereumProvider };
    okxwallet?: { ethereum?: EthereumProvider };
    bitkeep?: { ethereum?: EthereumProvider };
    ethereum?: EthereumProvider;
    phantom?: { ethereum?: EthereumProvider };
  };
  if (w.$onekey?.ethereum) return w.$onekey.ethereum;
  if (w.okxwallet?.ethereum) return w.okxwallet.ethereum;
  if (w.bitkeep?.ethereum) return w.bitkeep.ethereum;
  if (w.ethereum) return w.ethereum;
  if (w.phantom?.ethereum) return w.phantom.ethereum;
  return null;
}

export async function getEthereumAddress(): Promise<string> {
  const provider = getEthereumProvider();
  if (!provider) throw new Error("No Ethereum wallet detected");
  const accounts = await provider.request({ method: "eth_requestAccounts" });
  if (!accounts?.length) throw new Error("No accounts found. Please connect your wallet.");
  return accounts[0];
}

/**
 * Sign message for SIWE (EIP-191). Tries personal_sign first, falls back to eth_sign.
 * Many wallets support eth_sign with [address, message] (plain string) for compatibility.
 */
export async function signEthereumMessage(message: string, address: string): Promise<string> {
  const provider = getEthereumProvider();
  if (!provider) throw new Error("No Ethereum wallet detected");
  const messageHex =
    "0x" +
    Array.from(new TextEncoder().encode(message))
      .map((b) => b.toString(16).padStart(2, "0"))
      .join("");

  const handleSignError = (err: unknown): never => {
    const code = (err as { code?: number })?.code;
    if (code === 4001 || code === -32603) {
      throw new Error("User rejected the signature request");
    }
    throw new Error(`Failed to sign: ${err instanceof Error ? err.message : "Unknown error"}`);
  };

  // Try personal_sign first (standard EIP-191, params: [messageHex, address])
  try {
    const sig = await provider.request({
      method: "personal_sign",
      params: [messageHex, address],
    });
    return sig;
  } catch (err: unknown) {
    // Fall back to eth_sign [address, message] for wallets that don't support personal_sign
    try {
      const sig = await provider.request({
        method: "eth_sign",
        params: [address, message],
      });
      return sig;
    } catch (_err2) {
      handleSignError(err);
    }
  }
}

export function getEthereumChainId(): Promise<string> {
  const provider = getEthereumProvider();
  if (!provider) throw new Error("No Ethereum wallet detected");
  return provider.request({ method: "eth_chainId" });
}
