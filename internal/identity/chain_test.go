package identity

import (
	"math/big"
	"testing"
)

func TestCAIP2ToChainType(t *testing.T) {
	tests := []struct {
		caip2    string
		want     ChainType
		wantErr  bool
	}{
		{"eip155:8453", ChainTypeEthereum, false},
		{"eip155:1", ChainTypeEthereum, false},
		{"bip122:000000000019d6689c085ae165831e93", ChainTypeBitcoinSegwit, false},
		{"tron:728126428", ChainTypeTron, false},
		{"xrpl:0", ChainTypeXRP, false},
		{"sui:mainnet", ChainTypeSui, false},
		{"polkadot:91b171bb158e2d3848fa23a9f1c25182", ChainTypePolkadot, false},
		{"solana:5eykt4zzFbh8iLudsaDkvVygepqvV22oZwod8EqB", ChainTypeSolana, false},
		{"eip155:10900", ChainTypeMorpheum, false},
		{"unknown:123", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		got, err := CAIP2ToChainType(tt.caip2)
		if (err != nil) != tt.wantErr {
			t.Errorf("CAIP2ToChainType(%q) err = %v, wantErr %v", tt.caip2, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("CAIP2ToChainType(%q) = %q, want %q", tt.caip2, got, tt.want)
		}
	}
}

func TestValidateAddressForChain(t *testing.T) {
	validEVM := "0x0000000000000000000000000000000000000001"
	tests := []struct {
		addr    string
		caip2   string
		wantErr bool
	}{
		{validEVM, "eip155:8453", false},
		{"0x123", "eip155:1", true},
		{"bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh", "bip122:000000000019d6689c085ae165831e93", false},
		{"bc1p", "bip122:xxx", true},
		{"TXYZopYRdj2D9XRtbGxoX3bHd4rDQUc3fH", "tron:728126428", false},
		{"rN7n7otQDd6FczFgLdlqtyMVrn3e1Djxv7", "xrpl:0", false},
		{"0x0000000000000000000000000000000000000000000000000000000000000001", "sui:mainnet", false},
		{"mr4m1qxkunrkxrp90jte038d8ex82tdeqjedvvudecqtsypp49kvydel8vwcnx8r", "eip155:10900", false},
		{"0x0000000000000000000000000000000000000001", "eip155:10900", true},
		{"", "eip155:8453", true},
		{validEVM, "unknown:1", true},
	}
	for _, tt := range tests {
		err := ValidateAddressForChain(tt.addr, tt.caip2)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateAddressForChain(%q, %q) err = %v, wantErr %v", tt.addr, tt.caip2, err, tt.wantErr)
		}
	}
}

func TestAddressKeyForChain(t *testing.T) {
	if got := AddressKeyForChain("eip155:8453"); got != "address_eip155:8453" {
		t.Errorf("AddressKeyForChain(eip155:8453) = %q, want address_eip155:8453", got)
	}
	if got := AddressKeyForChain("bip122:xxx"); got != "address_bip122:xxx" {
		t.Errorf("AddressKeyForChain(bip122:xxx) = %q, want address_bip122:xxx", got)
	}
}

func TestDeriveAddress_NonEVM(t *testing.T) {
	_, err := DeriveAddress("bip122:000000000019d6689c085ae165831e93")
	if err == nil {
		t.Error("DeriveAddress(bip122) should error (non-EVM not implemented)")
	}
	_, err = DeriveAddress("unknown:1")
	if err == nil {
		t.Error("DeriveAddress(unknown) should error")
	}
}

func TestValidateChainNonce(t *testing.T) {
	if err := ValidateChainNonce("0x1234", "eip155:8453", big.NewInt(1)); err != nil {
		t.Errorf("ValidateChainNonce valid: %v", err)
	}
	if err := ValidateChainNonce("", "eip155:8453", big.NewInt(1)); err == nil {
		t.Error("ValidateChainNonce empty owner should error")
	}
	if err := ValidateChainNonce("0x1234", "eip155:8453", big.NewInt(-1)); err == nil {
		t.Error("ValidateChainNonce negative nonce should error")
	}
}

func TestNamespace(t *testing.T) {
	tests := []struct {
		caip2   string
		want    string
	}{
		{"eip155:8453", "eip155"},
		{"bip122:000000000019d6689c085ae165831e93", "bip122"},
		{"tron:728126428", "tron"},
		{"invalid", ""},
	}
	for _, tt := range tests {
		got := Namespace(tt.caip2)
		if got != tt.want {
			t.Errorf("Namespace(%q) = %q, want %q", tt.caip2, got, tt.want)
		}
	}
}

func TestChainIDFromCAIP2(t *testing.T) {
	tests := []struct {
		caip2 string
		want  uint64
	}{
		{"eip155:8453", 8453},
		{"eip155:1", 1},
		{"eip155:42161", 42161},
		{"bip122:xxx", 0},
		{"eip155:invalid", 0},
	}
	for _, tt := range tests {
		got := ChainIDFromCAIP2(tt.caip2)
		if got != tt.want {
			t.Errorf("ChainIDFromCAIP2(%q) = %d, want %d", tt.caip2, got, tt.want)
		}
	}
}

func TestChainIDToCAIP2(t *testing.T) {
	if got := ChainIDToCAIP2(8453); got != "eip155:8453" {
		t.Errorf("ChainIDToCAIP2(8453) = %q, want eip155:8453", got)
	}
}

func TestIsEVM(t *testing.T) {
	if !IsEVM("eip155:8453") {
		t.Error("eip155:8453 should be EVM")
	}
	if IsEVM("bip122:xxx") {
		t.Error("bip122 should not be EVM")
	}
}

func TestChainIDBig(t *testing.T) {
	got := ChainIDBig("eip155:8453")
	if got == nil || got.Cmp(big.NewInt(8453)) != 0 {
		t.Errorf("ChainIDBig(eip155:8453) = %v, want 8453", got)
	}
	if ChainIDBig("bip122:xxx") != nil {
		t.Error("non-EVM should return nil")
	}
}
