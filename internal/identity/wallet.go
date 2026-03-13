package identity

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

const walletFilename = "wallet.json"

// EVMAccount implements Account for EVM chains.
type EVMAccount struct {
	key     *ecdsa.PrivateKey
	address string
}

// Address returns the EVM address (0x-prefixed).
func (a *EVMAccount) Address() string {
	return a.address
}

// SignMessage signs a message with the account's private key (EIP-191 personal sign).
func (a *EVMAccount) SignMessage(message []byte) ([]byte, error) {
	prefix := []byte("\x19Ethereum Signed Message:\n" + fmt.Sprintf("%d", len(message)))
	hash := crypto.Keccak256Hash(append(prefix, message...))
	return crypto.Sign(hash.Bytes(), a.key)
}

// PrivateKey returns the raw private key for SIWE (do not expose to agent tools).
func (a *EVMAccount) PrivateKey() *ecdsa.PrivateKey {
	return a.key
}

// GetAutomatonDir returns ~/.automaton or AUTOMATON_DIR.
func GetAutomatonDir() string {
	if d := os.Getenv("AUTOMATON_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".automaton")
}

// GetWalletPath returns the full path to wallet.json.
func GetWalletPath() string {
	return filepath.Join(GetAutomatonDir(), walletFilename)
}

// GetWallet loads or creates the automaton wallet (EVM primary).
// Returns (account, isNew, error).
func GetWallet() (*EVMAccount, bool, error) {
	dir := GetAutomatonDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, false, fmt.Errorf("create automaton dir: %w", err)
	}
	path := GetWalletPath()

	if data, err := os.ReadFile(path); err == nil {
		var w WalletData
		if err := json.Unmarshal(data, &w); err != nil {
			return nil, false, fmt.Errorf("parse wallet: %w", err)
		}
		key, err := parsePrivateKey(w.PrivateKey)
		if err != nil {
			return nil, false, err
		}
		addr := crypto.PubkeyToAddress(key.PublicKey)
		return &EVMAccount{key: key, address: addr.Hex()}, false, nil
	}

	// Create new wallet
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, false, fmt.Errorf("generate key: %w", err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)
	privHex := "0x" + hex.EncodeToString(crypto.FromECDSA(key))
	w := WalletData{
		PrivateKey: privHex,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	data, _ := json.MarshalIndent(w, "", "  ")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return nil, false, fmt.Errorf("write wallet: %w", err)
	}
	return &EVMAccount{key: key, address: addr.Hex()}, true, nil
}

// GetWalletAddress returns the primary (EVM) address, or "" if no wallet.
func GetWalletAddress() string {
	path := GetWalletPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var w WalletData
	if err := json.Unmarshal(data, &w); err != nil {
		return ""
	}
	key, err := parsePrivateKey(w.PrivateKey)
	if err != nil {
		return ""
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)
	return addr.Hex()
}

// DeriveAddress returns the address for the given CAIP-2 chain.
// EVM (eip155): same secp256k1 address for all EVM chains; validates format before return.
// Non-EVM (bip122, tron, xrpl, sui, polkadot): returns error until per-chain derivation libs are added.
func DeriveAddress(chainCAIP2 string) (string, error) {
	if IsEVM(chainCAIP2) {
		addr := GetWalletAddress()
		if addr == "" {
			return "", fmt.Errorf("no wallet: run 'moneyclaw init' first")
		}
		if err := ValidateAddressForChain(addr, chainCAIP2); err != nil {
			return "", fmt.Errorf("validate EVM address: %w", err)
		}
		return addr, nil
	}
	// Non-EVM: requires chain-specific libs (btcutil, tron, ripple, sui-go-sdk, etc.)
	if _, err := CAIP2ToChainType(chainCAIP2); err != nil {
		return "", err
	}
	return "", fmt.Errorf("chain %s: non-EVM derivation not yet implemented (requires chain-specific libs)", chainCAIP2)
}

// WalletExists returns true if wallet.json exists.
func WalletExists() bool {
	_, err := os.Stat(GetWalletPath())
	return err == nil
}

func parsePrivateKey(s string) (*ecdsa.PrivateKey, error) {
	s = strings.TrimPrefix(s, "0x")
	bytes, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}
	return crypto.ToECDSA(bytes)
}
