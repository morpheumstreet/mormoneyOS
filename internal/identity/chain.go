package identity

import (
	"math/big"
	"strconv"
	"strings"
)

// DefaultChainBase is the default CAIP-2 chain (Base).
const DefaultChainBase = "eip155:8453"

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

// IsEVM returns true if the chain uses EIP-155 (Ethereum, Base, Polygon, etc.).
func IsEVM(caip2 string) bool {
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
