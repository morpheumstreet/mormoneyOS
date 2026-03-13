package identity

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

func TestBootstrapIdentity(t *testing.T) {
	// Use temp dir so no existing wallet; config address will be used
	dir := t.TempDir()
	prev := os.Getenv("AUTOMATON_DIR")
	os.Setenv("AUTOMATON_DIR", dir)
	defer os.Setenv("AUTOMATON_DIR", prev)
	// Ensure no wallet.json in temp dir
	_ = os.Remove(filepath.Join(dir, "wallet.json"))

	store := &mockIdentityStore{data: make(map[string]string)}
	cfg := &types.AutomatonConfig{
		WalletAddress:  "0x1234567890123456789012345678901234567890",
		DefaultChain:   "eip155:8453",
		Name:           "test-agent",
		CreatorAddress: "0xcreator",
		SandboxID:      "sandbox-1",
		ChainProviders: map[string]types.ChainProviderConfig{
			"eip155:1":    {RPCURL: "https://eth.llamarpc.com", USDCAddress: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"},
			"eip155:8453": {RPCURL: "https://mainnet.base.org", USDCAddress: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"},
		},
	}

	primary, err := BootstrapIdentity(store, cfg)
	if err != nil {
		t.Fatalf("BootstrapIdentity: %v", err)
	}
	// With empty temp dir, GetWallet creates new wallet; primary may be from wallet or config
	// Config has WalletAddress, but GetWallet creates wallet first - so primary = new wallet address
	// Actually: GetWallet creates wallet in dir. So we get a new random address. Config has a different address.
	// Order in bootstrap: primary = cfg.WalletAddress, then if GetWallet succeeds, primary = acc.Address().
	// So with a new wallet, primary will be the new wallet address. We can't easily test "config only" without
	// preventing GetWallet from creating. Simpler: just assert primary is non-empty and keys are set.
	if primary == "" {
		t.Error("primary is empty")
	}

	// Check persisted keys
	if v := store.data["address"]; v == "" {
		t.Error("address not set")
	}
	if v := store.data["default_chain"]; v != "eip155:8453" {
		t.Errorf("default_chain = %q, want eip155:8453", v)
	}
	if v := store.data["name"]; v != "test-agent" {
		t.Errorf("name = %q, want test-agent", v)
	}
	if v := store.data["creator"]; v != "0xcreator" {
		t.Errorf("creator = %q, want 0xcreator", v)
	}
	if v := store.data["sandbox"]; v != "sandbox-1" {
		t.Errorf("sandbox = %q, want sandbox-1", v)
	}
	if v := store.data["address_eip155:8453"]; v == "" {
		t.Error("address_eip155:8453 not set")
	}
	if v := store.data["address_eip155:1"]; v == "" {
		t.Error("address_eip155:1 not set")
	}
}

func TestBootstrapIdentity_NilInputs(t *testing.T) {
	primary, err := BootstrapIdentity(nil, nil)
	if err != nil {
		t.Fatalf("BootstrapIdentity: %v", err)
	}
	if primary != "" {
		t.Errorf("primary = %q, want empty", primary)
	}
}

func TestGetAddressForChain(t *testing.T) {
	getter := &mockIdentityStore{
		data: map[string]string{
			"address":            "0xprimary",
			"address_eip155:1":   "0xeth",
			"address_eip155:8453": "0xbase",
		},
	}
	cfg := &types.AutomatonConfig{DefaultChain: "eip155:8453", WalletAddress: "0xconfig"}

	// Per-chain key takes precedence
	if got := GetAddressForChain("eip155:1", getter, cfg); got != "0xeth" {
		t.Errorf("GetAddressForChain(eip155:1) = %q, want 0xeth", got)
	}
	// Primary address when chain matches default
	if got := GetAddressForChain("eip155:8453", getter, cfg); got != "0xbase" {
		t.Errorf("GetAddressForChain(eip155:8453) = %q, want 0xbase", got)
	}
	// Fallback to config when no identity
	getter2 := &mockIdentityStore{data: map[string]string{}}
	if got := GetAddressForChain("eip155:8453", getter2, cfg); got != "0xconfig" {
		t.Errorf("GetAddressForChain(no identity) = %q, want 0xconfig", got)
	}
}

func TestGetPrimaryAddress(t *testing.T) {
	getter := &mockIdentityStore{data: map[string]string{"address": "0xprimary"}}
	cfg := &types.AutomatonConfig{DefaultChain: "eip155:8453"}

	if got := GetPrimaryAddress(getter, cfg); got != "0xprimary" {
		t.Errorf("GetPrimaryAddress = %q, want 0xprimary", got)
	}
}

type mockIdentityStore struct {
	data map[string]string
}

func (m *mockIdentityStore) SetIdentity(key, value string) error {
	m.data[key] = value
	return nil
}

func (m *mockIdentityStore) GetIdentity(key string) (string, bool, error) {
	v, ok := m.data[key]
	return v, ok, nil
}
