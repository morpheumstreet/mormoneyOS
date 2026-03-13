// Package conway: USDC balance check via chain providers (Base RPC eth_call).
// Supports multiple chains via configurable providers (rpcUrl + usdcAddress per chain).
package conway

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// USDCChainProvider configures RPC and USDC contract for a chain (CAIP-2, e.g. eip155:8453).
type USDCChainProvider struct {
	RPCURL      string `json:"rpcUrl"`      // e.g. https://mainnet.base.org
	USDCAddress string `json:"usdcAddress"`  // USDC contract, e.g. 0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913
}

// BalanceResult holds USDC balance for one chain.
type BalanceResult struct {
	Chain   string  `json:"chain"`
	Balance float64 `json:"balance"` // USD (6 decimals)
}

// DefaultChainProviders are built-in providers for known USDC chains.
var DefaultChainProviders = map[string]USDCChainProvider{
	"eip155:8453":  {RPCURL: "https://mainnet.base.org", USDCAddress: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"},   // Base mainnet
	"eip155:84532": {RPCURL: "https://sepolia.base.org", USDCAddress: "0x036CbD53842c5426634e7929541eC2318f3dCF7e"},   // Base Sepolia
	"eip155:1":     {RPCURL: "https://eth.llamarpc.com", USDCAddress: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"},   // Ethereum mainnet
	"eip155:137":   {RPCURL: "https://polygon-rpc.com", USDCAddress: "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359"},   // Polygon (native USDC)
	"eip155:42161": {RPCURL: "https://arb1.arbitrum.io/rpc", USDCAddress: "0xaf88d065e77c8cC2239327C5B04290e6Abe2A490"}, // Arbitrum One
}

// GetUSDCBalance returns USDC balance for address on the given chain.
// Uses provider if non-nil; otherwise DefaultChainProviders for known chains.
// Returns balance in USD (6 decimals -> float).
func GetUSDCBalance(ctx context.Context, address string, chain string, provider *USDCChainProvider) (float64, error) {
	if chain == "" {
		chain = "eip155:8453"
	}
	p := provider
	if p == nil {
		if def, ok := DefaultChainProviders[chain]; ok {
			p = &def
		} else {
			return 0, fmt.Errorf("unsupported USDC chain: %s (no provider)", chain)
		}
	}
	if p.RPCURL == "" || p.USDCAddress == "" {
		return 0, fmt.Errorf("invalid provider for chain %s: rpcUrl and usdcAddress required", chain)
	}
	raw, err := callUSDCBalanceOfProvider(ctx, p.RPCURL, p.USDCAddress, address)
	if err != nil {
		return 0, err
	}
	return float64(raw) / 1_000_000, nil
}

// GetUSDCBalanceMulti returns USDC balances for address across multiple chains.
// providers overrides DefaultChainProviders for given chains; nil uses built-in only.
func GetUSDCBalanceMulti(ctx context.Context, address string, chains []string, providers map[string]USDCChainProvider) ([]BalanceResult, error) {
	if len(chains) == 0 {
		chains = []string{"eip155:8453"}
	}
	var out []BalanceResult
	for _, chain := range chains {
		var p *USDCChainProvider
		if providers != nil {
			if def, ok := providers[chain]; ok {
				p = &def
			}
		}
		if p == nil {
			if def, ok := DefaultChainProviders[chain]; ok {
				p = &def
			}
		}
		if p == nil {
			continue
		}
		bal, err := GetUSDCBalance(ctx, address, chain, p)
		if err != nil {
			continue
		}
		out = append(out, BalanceResult{Chain: chain, Balance: bal})
	}
	return out, nil
}

// callUSDCBalanceOfProvider performs eth_call to USDC balanceOf(account).
func callUSDCBalanceOfProvider(ctx context.Context, rpcURL, usdcAddr, account string) (int64, error) {
	data := "0x70a08231" + strings.Repeat("0", 24) + strings.TrimPrefix(common.HexToAddress(account).Hex(), "0x")
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_call",
		"params": []interface{}{
			map[string]string{"to": usdcAddr, "data": data},
			"latest",
		},
	}
	var rpcResp struct {
		Result string `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := ethCall(ctx, rpcURL, payload, &rpcResp); err != nil {
		return 0, err
	}
	if rpcResp.Error != nil {
		return 0, fmt.Errorf("rpc error: %s", rpcResp.Error.Message)
	}
	hexVal := strings.TrimPrefix(rpcResp.Result, "0x")
	if hexVal == "" {
		return 0, nil
	}
	b := new(big.Int)
	if _, ok := b.SetString(hexVal, 16); !ok {
		return 0, fmt.Errorf("invalid hex balance: %s", rpcResp.Result)
	}
	if !b.IsInt64() {
		return b.Int64(), nil
	}
	return b.Int64(), nil
}
