package identity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeriveAddressFromMnemonic_EVM(t *testing.T) {
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	addr, err := DeriveAddressFromMnemonic(mnemonic, "eip155:8453", 0)
	if err != nil {
		t.Fatalf("DeriveAddressFromMnemonic: %v", err)
	}
	if !strings.HasPrefix(addr, "0x") || len(addr) != 42 {
		t.Errorf("expected 0x-prefixed 42-char EVM address, got %q", addr)
	}
}

func TestDeriveAddressFromMnemonic_Bitcoin(t *testing.T) {
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	addr, err := DeriveAddressFromMnemonic(mnemonic, "bip122:000000000019d6689c085ae165831e93", 0)
	if err != nil {
		t.Fatalf("DeriveAddressFromMnemonic: %v", err)
	}
	if !strings.HasPrefix(addr, "bc1") {
		t.Errorf("expected bc1-prefixed Bitcoin address, got %q", addr)
	}
}

func TestDeriveAddressFromMnemonic_Solana(t *testing.T) {
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	addr, err := DeriveAddressFromMnemonic(mnemonic, "solana:5eykt4zzFbh8iLudsaDkvVygepqvV22oZwod8EqB", 0)
	if err != nil {
		t.Fatalf("DeriveAddressFromMnemonic: %v", err)
	}
	if len(addr) < 32 || len(addr) > 44 {
		t.Errorf("expected Solana address 32-44 chars, got %q (len=%d)", addr, len(addr))
	}
}

func TestDeriveAddressFromMnemonic_Morpheum(t *testing.T) {
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	addr, err := DeriveAddressFromMnemonic(mnemonic, MorpheumTestnetCAIP2, 0)
	if err != nil {
		t.Fatalf("DeriveAddressFromMnemonic: %v", err)
	}
	if !strings.HasPrefix(addr, "mr4m") {
		t.Errorf("expected mr4m-prefixed Morpheum address, got %q", addr)
	}
}

func TestDeriveAddressFromMnemonic_Unsupported(t *testing.T) {
	_, err := DeriveAddressFromMnemonic("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "tron:728126428", 0)
	if err == nil {
		t.Error("expected error for unsupported tron chain")
	}
}

func TestGetWallet_NewMnemonicWallet(t *testing.T) {
	dir := t.TempDir()
	prev := os.Getenv("AUTOMATON_DIR")
	os.Setenv("AUTOMATON_DIR", dir)
	defer os.Setenv("AUTOMATON_DIR", prev)

	acc, isNew, err := GetWallet()
	if err != nil {
		t.Fatalf("GetWallet: %v", err)
	}
	if !isNew {
		t.Error("expected isNew=true for fresh dir")
	}
	if acc.Address() == "" || !strings.HasPrefix(acc.Address(), "0x") {
		t.Errorf("expected 0x EVM address, got %q", acc.Address())
	}

	// Verify wallet.json has mnemonic only (no privateKey)
	data, err := os.ReadFile(filepath.Join(dir, "wallet.json"))
	if err != nil {
		t.Fatalf("read wallet: %v", err)
	}
	if !strings.Contains(string(data), `"mnemonic"`) {
		t.Error("expected mnemonic in wallet.json")
	}
	if strings.Contains(string(data), `"privateKey"`) {
		t.Error("wallet must be mnemonic-only; no privateKey")
	}
}
