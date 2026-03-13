package identity

import (
	"fmt"
	"strings"

	stdvalidation "github.com/morpheum-labs/standards/clitool/validation"
	"github.com/morpheum-labs/standards/types"
)

// AddressValidator validates address format for a specific chain type.
// SOLID: Interface Segregation — narrow, single-purpose contract.
type AddressValidator interface {
	Validate(addr string) error
}

// chainValidators holds per-chain validators. Populated at init.
// SOLID: Open/Closed — add new chain by registering, no switch modification.
var chainValidators = make(map[ChainType]AddressValidator)

func init() {
	registerChainValidators()
}

func registerChainValidators() {
	chainValidators[ChainTypeEthereum] = hexValidator{totalLen: 42, name: "EVM"}
	chainValidators[ChainTypeBitcoinSegwit] = prefixValidator{prefixes: []string{"bc1q", "tb1q"}, minLen: 14, name: "Bitcoin SegWit"}
	chainValidators[ChainTypeBitcoinTaproot] = prefixValidator{prefixes: []string{"bc1p", "tb1p"}, minLen: 14, name: "Bitcoin Taproot"}
	chainValidators[ChainTypeBitcoinLegacy] = firstCharValidator{chars: "1mn", minLen: 26, name: "Bitcoin Legacy"}
	chainValidators[ChainTypeBitcoinNested] = firstCharValidator{chars: "32", minLen: 26, name: "Bitcoin Nested SegWit"}
	chainValidators[ChainTypeTron] = tronValidator{}
	chainValidators[ChainTypeXRP] = prefixValidator{prefixes: []string{"r", "X"}, minLen: 25, name: "XRP"}
	chainValidators[ChainTypeSui] = hexValidator{totalLen: 66, name: "Sui"}
	chainValidators[ChainTypeSolana] = lengthRangeValidator{minLen: 32, maxLen: 44, name: "Solana"}
	chainValidators[ChainTypePolkadot] = polkadotValidator{}
}

// hexValidator validates 0x-prefixed hex addresses (EVM, Sui).
// DRY: Shared for EVM (42 chars) and Sui (66 chars).
type hexValidator struct {
	totalLen int
	name     string
}

func (v hexValidator) Validate(addr string) error {
	return validateHexAddress(addr, v.totalLen, v.name)
}

// prefixValidator validates addresses with allowed prefixes (Bitcoin variants, XRP).
type prefixValidator struct {
	prefixes []string
	minLen   int
	name     string
}

func (v prefixValidator) Validate(addr string) error {
	return validatePrefixBased(addr, v.prefixes, v.minLen, v.name)
}

// firstCharValidator validates by first character (Bitcoin Legacy, Nested).
type firstCharValidator struct {
	chars   string
	minLen  int
	name    string
}

func (v firstCharValidator) Validate(addr string) error {
	return validateFirstChar(addr, v.chars, v.minLen, v.name)
}

// lengthRangeValidator validates by length range only (Solana).
type lengthRangeValidator struct {
	minLen, maxLen int
	name          string
}

func (v lengthRangeValidator) Validate(addr string) error {
	if len(addr) < v.minLen || len(addr) > v.maxLen {
		return fmt.Errorf("invalid %s address: expected %d-%d chars, got %d", v.name, v.minLen, v.maxLen, len(addr))
	}
	return nil
}

// tronValidator validates Tron addresses (T or 27 prefix, 34-50 chars).
type tronValidator struct{}

func (tronValidator) Validate(addr string) error {
	if len(addr) < 34 {
		return fmt.Errorf("invalid Tron address: too short")
	}
	if len(addr) > 50 {
		return fmt.Errorf("invalid Tron address: too long")
	}
	if strings.HasPrefix(addr, "T") || strings.HasPrefix(addr, "27") {
		return nil
	}
	return fmt.Errorf("invalid Tron address: expected T (mainnet) or 27 (testnet) prefix")
}

// polkadotValidator validates Polkadot SS58 addresses.
type polkadotValidator struct{}

func (polkadotValidator) Validate(addr string) error {
	if len(addr) < 47 {
		return fmt.Errorf("invalid Polkadot address: too short")
	}
	if (addr[0] >= '1' && addr[0] <= '9') || addr[0] == 'F' {
		return nil
	}
	return fmt.Errorf("invalid Polkadot address: expected SS58 format")
}

// validateHexAddress — DRY helper for 0x-prefixed hex (EVM, Sui).
func validateHexAddress(addr string, totalLen int, chainName string) error {
	if len(addr) != totalLen || !strings.HasPrefix(addr, "0x") {
		return fmt.Errorf("invalid %s address: expected 0x + %d hex chars, got %d chars", chainName, totalLen-2, len(addr))
	}
	for i := 2; i < len(addr); i++ {
		c := addr[i]
		if !isHexChar(c) {
			return fmt.Errorf("invalid %s address: non-hex character at position %d", chainName, i)
		}
	}
	return nil
}

func isHexChar(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// validatePrefixBased — DRY helper for prefix-based validation.
func validatePrefixBased(addr string, prefixes []string, minLen int, chainName string) error {
	if len(addr) < minLen {
		return fmt.Errorf("invalid %s address: too short", chainName)
	}
	for _, p := range prefixes {
		if strings.HasPrefix(addr, p) {
			return nil
		}
	}
	return fmt.Errorf("invalid %s address: expected one of %v", chainName, prefixes)
}

// validateFirstChar — DRY helper for first-character validation.
func validateFirstChar(addr string, allowedChars string, minLen int, chainName string) error {
	if len(addr) < minLen {
		return fmt.Errorf("invalid %s address: too short", chainName)
	}
	for i := 0; i < len(allowedChars); i++ {
		if addr[0] == allowedChars[i] {
			return nil
		}
	}
	return fmt.Errorf("invalid %s address: expected first char in %q", chainName, allowedChars)
}

// chainTypeToStandards maps mormoneyOS ChainType to standards types.ChainType.
// Returns (standards type, true) when standards supports the chain; (_, false) otherwise.
func chainTypeToStandards(ct ChainType) (types.ChainType, bool) {
	switch ct {
	case ChainTypeEthereum:
		return types.ChainTypeEthereum, true
	case ChainTypeBitcoinSegwit:
		return types.ChainTypeBitcoinSegwit, true
	case ChainTypeBitcoinTaproot:
		return types.ChainTypeBitcoinTaproot, true
	case ChainTypeBitcoinLegacy:
		return types.ChainTypeBitcoinLegacy, true
	case ChainTypeBitcoinNested:
		return types.ChainTypeBitcoinNestedSegwit, true
	case ChainTypeSolana:
		return types.ChainTypeSolana, true
	case ChainTypeMorpheum:
		return types.ChainTypeMorpheum, true
	default:
		return "", false
	}
}

// validateAddressFormat delegates to standards when supported (EVM, Bitcoin, Solana, Morpheum),
// otherwise uses local chain validator registry (Tron, XRP, Sui, Polkadot).
func validateAddressFormat(addr string, ct ChainType) error {
	if stdCT, ok := chainTypeToStandards(ct); ok {
		return stdvalidation.ValidateAddressByChain(addr, stdCT)
	}
	v, ok := chainValidators[ct]
	if !ok {
		return fmt.Errorf("unsupported chain type: %s", ct)
	}
	return v.Validate(addr)
}
