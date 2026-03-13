// Package identity: multi-chain address bootstrap and resolver.
package identity

import (
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// IdentityStore is the minimal interface for identity bootstrap (persist keys).
type IdentityStore interface {
	SetIdentity(key, value string) error
}

// BootstrapIdentity derives and persists multi-chain addresses for all configured chains.
// Prefers live wallet when available; falls back to config.WalletAddress.
// Stores: address (primary for default chain), default_chain, address_<caip2> per chain,
// and TS-aligned keys: name, creator, sandbox, createdAt (createdAt never overwritten).
// Returns the primary address (for default chain) for use by daemon/loop.
func BootstrapIdentity(store IdentityStore, cfg *types.AutomatonConfig) (primaryAddress string, err error) {
	if store == nil || cfg == nil {
		return "", nil
	}

	// Primary address: live wallet first, else config
	primary := cfg.WalletAddress
	if acc, _, e := GetWallet(); e == nil && acc != nil && acc.Address() != "" {
		primary = acc.Address()
	}
	if primary == "" && cfg.DefaultChain != "" {
		if addr, e := DeriveAddress(cfg.DefaultChain); e == nil && addr != "" {
			primary = addr
		}
	}

	// Persist primary and default chain
	_ = store.SetIdentity("address", primary)
	_ = store.SetIdentity("default_chain", cfg.DefaultChain)

	// Collect all chains to bootstrap: defaultChain + chainProviders
	chains := make(map[string]struct{})
	if cfg.DefaultChain != "" {
		chains[cfg.DefaultChain] = struct{}{}
	}
	for c := range cfg.ChainProviders {
		if c != "" {
			chains[c] = struct{}{}
		}
	}
	// Fallback when no config
	if len(chains) == 0 && cfg.DefaultChain == "" {
		chains[DefaultChainBase] = struct{}{}
	}

	// Derive and store address per chain
	for chain := range chains {
		addr, e := DeriveAddress(chain)
		if e != nil || addr == "" {
			// Non-EVM or derivation failed: use primary if EVM default
			if primary != "" && IsEVM(chain) {
				addr = primary
			} else if primary != "" {
				addr = primary
			}
		}
		if addr != "" {
			_ = store.SetIdentity(AddressKeyForChain(chain), addr)
		}
	}

	// TS-aligned identity keys (name, creator, sandbox)
	if cfg.Name != "" {
		_ = store.SetIdentity("name", cfg.Name)
	}
	if cfg.CreatorAddress != "" {
		_ = store.SetIdentity("creator", cfg.CreatorAddress)
	}
	if cfg.SandboxID != "" {
		_ = store.SetIdentity("sandbox", cfg.SandboxID)
	}
	// createdAt: only set if not already present (never overwrite)
	// Caller should check GetIdentity("createdAt") and only set when missing
	// We don't have GetIdentity in this interface; caller handles createdAt in run.go if needed

	return primary, nil
}
