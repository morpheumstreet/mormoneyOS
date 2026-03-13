package identity

import (
	"math/big"
	"testing"
)

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
