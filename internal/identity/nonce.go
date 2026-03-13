package identity

import (
	"fmt"
	"math/big"
)

// ChainNonce represents a nonce scoped by (owner address, chain ID) for replay prevention.
// Per standards: nonces are per (owner, chainID), not global, to prevent cross-chain replay.
type ChainNonce struct {
	Owner   string   // Address (format depends on chain)
	ChainID *big.Int // EIP-155 chain ID for EVM; nil or chain-specific ID for others
	Nonce   *big.Int
}

// ValidateChainNonce checks that the nonce is valid for the given owner and chain.
// Used when validating signed operations to ensure replay protection.
// chainCAIP2 is the CAIP-2 chain identifier (e.g. eip155:8453).
func ValidateChainNonce(owner string, chainCAIP2 string, nonce *big.Int) error {
	if nonce == nil || nonce.Sign() < 0 {
		return fmt.Errorf("invalid nonce: must be non-negative")
	}
	if owner == "" {
		return fmt.Errorf("owner cannot be empty")
	}
	// Chain-scoped: for EVM, derive chainID from CAIP-2
	if IsEVM(chainCAIP2) {
		chainID := ChainIDFromCAIP2(chainCAIP2)
		if chainID == 0 {
			return fmt.Errorf("invalid EVM chain: %s", chainCAIP2)
		}
		_ = chainID // Caller should persist/validate (owner, chainID, nonce) tuple
	}
	return nil
}
