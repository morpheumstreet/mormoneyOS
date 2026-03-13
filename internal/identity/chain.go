package identity

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

// DefaultChainBase is the default CAIP-2 chain (Base).
const DefaultChainBase = "eip155:8453"

// ChainType represents the blockchain network type for multi-chain support.
// Local type for mormoneyOS; can align with standards types.ChainType when that package is used.
type ChainType string

const (
	ChainTypeEthereum       ChainType = "ethereum"
	ChainTypeBitcoinSegwit  ChainType = "bitcoin_segwit"
	ChainTypeBitcoinTaproot ChainType = "bitcoin_taproot"
	ChainTypeBitcoinLegacy  ChainType = "bitcoin_legacy"
	ChainTypeBitcoinNested  ChainType = "bitcoin_nested_segwit"
	ChainTypeTron           ChainType = "tron"
	ChainTypeXRP            ChainType = "xrpl"
	ChainTypeSui            ChainType = "sui"
	ChainTypeSolana         ChainType = "solana"
	ChainTypePolkadot       ChainType = "polkadot"
	ChainTypeMorpheum       ChainType = "morpheum" // Morpheum (eip155:10900) — mr4m addresses, ECDSA+ML-DSA-44
)

// MorpheumTestnetCAIP2 is the CAIP-2 for Morpheum testnet (chain ID 10900).
const MorpheumTestnetCAIP2 = "eip155:10900"

// ErrUnsupportedChain is returned when chain type is not supported.
var ErrUnsupportedChain = errors.New("unsupported chain type")

// CAIP2ToChainType maps CAIP-2 to ChainType for validation/signing.
// eip155:10900 -> morpheum; eip155:* -> ethereum; bip122:* -> bitcoin_segwit (default); tron:* -> tron; xrpl:* -> xrpl; sui:* -> sui; polkadot:* -> polkadot; solana:* -> solana.
func CAIP2ToChainType(caip2 string) (ChainType, error) {
	// Morpheum uses eip155:10900 (testnet) but mr4m addresses, not EVM.
	if caip2 == MorpheumTestnetCAIP2 {
		return ChainTypeMorpheum, nil
	}
	ns := Namespace(caip2)
	switch ns {
	case "eip155":
		return ChainTypeEthereum, nil
	case "bip122":
		return ChainTypeBitcoinSegwit, nil // default; can refine by reference
	case "tron":
		return ChainTypeTron, nil
	case "xrpl":
		return ChainTypeXRP, nil
	case "sui":
		return ChainTypeSui, nil
	case "polkadot":
		return ChainTypePolkadot, nil
	case "solana":
		return ChainTypeSolana, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedChain, ns)
	}
}

// ValidateAddressForChain validates address format for the given CAIP-2 chain.
// Performs format checks only (no full checksum validation for non-EVM).
// Delegates to chain-specific validators (validation.go).
func ValidateAddressForChain(addr, chainCAIP2 string) error {
	if addr == "" {
		return fmt.Errorf("address cannot be empty")
	}
	ct, err := CAIP2ToChainType(chainCAIP2)
	if err != nil {
		return err
	}
	return validateAddressFormat(addr, ct)
}

// AddressKeyForChain returns the identity table key for per-chain address: "address_<caip2>".
// Use for SetIdentity/GetIdentity.
func AddressKeyForChain(caip2 string) string {
	return "address_" + caip2
}

// Namespace returns the CAIP-2 namespace (e.g. "eip155", "bip122", "tron").
func Namespace(caip2 string) string {
	i := strings.Index(caip2, ":")
	if i < 0 {
		return ""
	}
	return caip2[:i]
}

// ChainIDFromCAIP2 parses "eip155:8453" -> 8453. Returns 0 if invalid or non-eip155.
func ChainIDFromCAIP2(caip2 string) uint64 {
	if Namespace(caip2) != "eip155" {
		return 0
	}
	i := strings.Index(caip2, ":")
	if i < 0 || i+1 >= len(caip2) {
		return 0
	}
	id, err := strconv.ParseUint(caip2[i+1:], 10, 64)
	if err != nil {
		return 0
	}
	return id
}

// ChainIDToCAIP2 formats 8453 -> "eip155:8453" (EVM only).
func ChainIDToCAIP2(chainID uint64) string {
	return "eip155:" + strconv.FormatUint(chainID, 10)
}

// IsEVM returns true if the chain uses EIP-155 and EVM-style 0x addresses.
// Morpheum (eip155:10900) uses mr4m addresses, not EVM.
func IsEVM(caip2 string) bool {
	if caip2 == MorpheumTestnetCAIP2 {
		return false
	}
	return Namespace(caip2) == "eip155"
}

// ChainIDBig returns the chain ID as *big.Int for EIP-712 domain. Nil if non-EVM.
func ChainIDBig(caip2 string) *big.Int {
	id := ChainIDFromCAIP2(caip2)
	if id == 0 {
		return nil
	}
	return new(big.Int).SetUint64(id)
}
