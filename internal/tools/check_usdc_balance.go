package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/conway"
	"github.com/morpheumlabs/mormoneyos-go/internal/identity"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// CheckUSDCBalanceTool checks USDC balance across configured chains (chain providers).
// Uses identity resolver when IdentityGetter is set (multi-chain address support).
type CheckUSDCBalanceTool struct {
	Config         *types.AutomatonConfig
	IdentityGetter identity.IdentityGetter // Optional; when set, resolves address from identity table
}

func (CheckUSDCBalanceTool) Name() string        { return "check_usdc_balance" }
func (CheckUSDCBalanceTool) Description() string { return "Check USDC balance across configured chains (Base, Ethereum, Polygon, etc.). Uses chainProviders from config when set." }
func (CheckUSDCBalanceTool) Parameters() string {
	return `{"type":"object","properties":{"chain":{"type":"string","description":"Optional CAIP-2 chain (e.g. eip155:8453). If omitted, uses defaultChain or all configured chainProviders."}},"required":[]}`
}

func (t *CheckUSDCBalanceTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	chains := []string{"eip155:8453"}
	var providers map[string]conway.USDCChainProvider

	if t.Config != nil {
		if t.Config.DefaultChain != "" {
			chains = []string{t.Config.DefaultChain}
		}
		if len(t.Config.ChainProviders) > 0 {
			providers = make(map[string]conway.USDCChainProvider)
			chains = make([]string, 0, len(t.Config.ChainProviders))
			for chain, cfg := range t.Config.ChainProviders {
				providers[chain] = conway.USDCChainProvider{RPCURL: cfg.RPCURL, USDCAddress: cfg.USDCAddress}
				chains = append(chains, chain)
			}
		}
	}
	chainArg := ""
	if c, ok := args["chain"].(string); ok && strings.TrimSpace(c) != "" {
		chainArg = strings.TrimSpace(c)
		chains = []string{chainArg}
		providers = nil
	}

	// Resolve address: identity table (multi-chain) first, else config
	address := ""
	if t.IdentityGetter != nil {
		if chainArg != "" {
			address = identity.GetAddressForChain(chainArg, t.IdentityGetter, t.Config)
		} else {
			address = identity.GetPrimaryAddress(t.IdentityGetter, t.Config)
		}
	}
	if address == "" && t.Config != nil {
		address = t.Config.WalletAddress
	}
	if address == "" {
		return "No wallet address. Run 'moneyclaw setup' and set walletAddress.", nil
	}

	results, err := conway.GetUSDCBalanceMulti(ctx, address, chains, providers)
	if err != nil {
		return "", fmt.Errorf("USDC balance check: %w", err)
	}
	if len(results) == 0 {
		return `{"total":0,"byChain":{},"message":"No chains checked (unsupported or missing provider)"}`, nil
	}
	var total float64
	byChain := make(map[string]float64)
	for _, r := range results {
		total += r.Balance
		byChain[r.Chain] = r.Balance
	}
	out := map[string]any{"total": total, "byChain": byChain}
	b, _ := json.Marshal(out)
	return string(b), nil
}
