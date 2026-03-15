// Package identity: multi-chain address derivation from mnemonic.
// Uses github.com/morpheum-labs/standards MultiChainKeyManager.
package identity

import (
	"crypto/ecdsa"
	"fmt"
	"strings"

	"github.com/morpheum-labs/standards/clitool"
	"github.com/morpheum-labs/standards/types"
)

// caip2ToStandardsChainType maps CAIP-2 to standards types.ChainType.
// Returns (chainType, isMainnet, error).
// Supported: eip155 (EVM), eip155:10900 (Morpheum), bip122 (Bitcoin), solana.
func caip2ToStandardsChainType(caip2 string) (types.ChainType, bool, error) {
	ns := Namespace(caip2)
	switch ns {
	case "eip155":
		if caip2 == MorpheumTestnetCAIP2 {
			return types.ChainTypeMorpheum, false, nil
		}
		return types.ChainTypeEthereum, true, nil
	case "bip122":
		// Bitcoin mainnet genesis: 000000000019d6689c085ae165831e93
		// Testnet: 000000000933ea01ad0ee984209779ba
		isMainnet := !strings.Contains(caip2, "000000000933ea01ad0ee984209779ba")
		return types.ChainTypeBitcoinSegwit, isMainnet, nil
	case "solana":
		return types.ChainTypeSolana, true, nil
	default:
		return "", false, fmt.Errorf("%w: %s", ErrUnsupportedChain, ns)
	}
}

// DeriveAddressFromMnemonic derives the address for the given CAIP-2 chain from mnemonic.
// Uses standards MultiChainKeyManager. Private keys are never persisted.
// index is the HD account index (BIP-44 path component).
func DeriveAddressFromMnemonic(mnemonic, chainCAIP2 string, index uint32) (string, error) {
	if mnemonic == "" {
		return "", fmt.Errorf("mnemonic required")
	}
	chainType, isMainnet, err := caip2ToStandardsChainType(chainCAIP2)
	if err != nil {
		return "", err
	}

	km := clitool.NewMultiChainKeyManager()
	key, err := km.DeriveKey(mnemonic, chainType, index)
	if err != nil {
		return "", fmt.Errorf("derive key for %s: %w", chainCAIP2, err)
	}

	addr, err := key.PublicKey().Address()
	if err != nil {
		return "", fmt.Errorf("derive address for %s: %w", chainCAIP2, err)
	}

	// Bitcoin keys from standards use mainnet by default in derivation.
	// ChainTypeBitcoinSegwit/Taproot etc. encode mainnet in the key.
	// The Address() already returns the correct format. isMainnet is used
	// by some derivation paths; MultiChainKeyManager uses mainnet=true for
	// Bitcoin by default. We pass through.
	_ = isMainnet

	return addr.String(), nil
}

// DeriveEVMPrivateKeyFromMnemonic derives ECDSA private key for EVM chains.
// Used by GetWallet when wallet is mnemonic-based.
// index is the HD account index (BIP-44 path component).
func DeriveEVMPrivateKeyFromMnemonic(mnemonic string, index uint32) (*ecdsa.PrivateKey, string, error) {
	return clitool.DeriveWalletFromMnemonic(mnemonic, index)
}
