// Package identity: multi-chain address resolution.
package identity

import (
	"time"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// IdentityGetter reads values from the identity store.
type IdentityGetter interface {
	GetIdentity(key string) (string, bool, error)
}

// GetAddressForChain returns the address for the given CAIP-2 chain.
// Resolution order: identity table (address_<caip2>) → identity "address" when chain is default →
// config WalletAddress → derive from wallet.
// Returns "" if no address can be resolved.
func GetAddressForChain(chain string, getter IdentityGetter, cfg *types.AutomatonConfig) string {
	if chain == "" && cfg != nil {
		chain = cfg.DefaultChain
	}
	if chain == "" {
		chain = DefaultChainBase
	}

	// 1. Per-chain key in identity table
	if getter != nil {
		if a, ok, _ := getter.GetIdentity(AddressKeyForChain(chain)); ok && a != "" {
			return a
		}
		// 2. Primary "address" key (for default chain)
		if a, ok, _ := getter.GetIdentity("address"); ok && a != "" {
			return a
		}
	}

	// 3. Config fallback
	if cfg != nil && cfg.WalletAddress != "" {
		return cfg.WalletAddress
	}

	// 4. Derive from wallet
	if addr, err := DeriveAddress(chain); err == nil && addr != "" {
		return addr
	}

	return ""
}

// GetPrimaryAddress returns the primary address (for default chain).
// Shorthand for GetAddressForChain with default chain.
func GetPrimaryAddress(getter IdentityGetter, cfg *types.AutomatonConfig) string {
	chain := DefaultChainBase
	if cfg != nil && cfg.DefaultChain != "" {
		chain = cfg.DefaultChain
	}
	return GetAddressForChain(chain, getter, cfg)
}

// EnsureCreatedAt sets identity "createdAt" only if not already present.
// Call from run init to align with TS behavior.
func EnsureCreatedAt(store interface {
	GetIdentity(key string) (string, bool, error)
	SetIdentity(key, value string) error
}) {
	if store == nil {
		return
	}
	if _, ok, _ := store.GetIdentity("createdAt"); ok {
		return
	}
	_ = store.SetIdentity("createdAt", time.Now().UTC().Format(time.RFC3339))
}
