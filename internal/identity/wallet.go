package identity

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/morpheum-labs/standards/clitool"
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

// loadWalletData reads and parses wallet.json. Returns nil if file does not exist.
func loadWalletData() (*WalletData, error) {
	path := GetWalletPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read wallet: %w", err)
	}
	var w WalletData
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("parse wallet: %w", err)
	}
	return &w, nil
}

// validateMnemonic validates mnemonic on load. Rejects corrupted files early.
func validateMnemonic(mnemonic string, wordCount int) error {
	if err := clitool.ValidateMnemonic(mnemonic); err != nil {
		return fmt.Errorf("invalid mnemonic: %w", err)
	}
	if wordCount > 0 {
		words := len(strings.Fields(mnemonic))
		if words != wordCount {
			return fmt.Errorf("mnemonic word count %d does not match stored wordCount %d", words, wordCount)
		}
	}
	return nil
}

// getAccountIndex returns the HD account index from wallet data. Default 0.
func getAccountIndex(w *WalletData) uint32 {
	if w == nil {
		return 0
	}
	return w.HDAccountIndex
}

// GetWallet loads or creates the automaton wallet (mnemonic-only).
// Returns (account, isNew, error).
func GetWallet() (*EVMAccount, bool, error) {
	dir := GetAutomatonDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, false, fmt.Errorf("create automaton dir: %w", err)
	}
	path := GetWalletPath()

	w, err := loadWalletData()
	if err != nil {
		return nil, false, err
	}

	if w != nil {
		mnemonic := strings.TrimSpace(w.Mnemonic)
		if mnemonic == "" {
			return nil, false, fmt.Errorf("wallet.json has no mnemonic; run 'moneyclaw init' to create a new mnemonic wallet")
		}
		if err := validateMnemonic(mnemonic, w.WordCount); err != nil {
			return nil, false, err
		}
		index := getAccountIndex(w)
		privKey, addr, err := getOrDeriveEVM(mnemonic, index)
		if err != nil {
			return nil, false, fmt.Errorf("derive from mnemonic: %w", err)
		}
		return &EVMAccount{key: privKey, address: addr}, false, nil
	}

	// Create new mnemonic wallet
	mnemonic, err := clitool.GenerateMnemonicFromLength(12)
	if err != nil {
		return nil, false, fmt.Errorf("generate mnemonic: %w", err)
	}
	wordCount := 12
	newW := WalletData{
		Mnemonic:       mnemonic,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		HDAccountIndex: 0,
		WordCount:      wordCount,
	}
	data, _ := json.MarshalIndent(newW, "", "  ")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return nil, false, fmt.Errorf("write wallet: %w", err)
	}
	privKey, addr, err := getOrDeriveEVM(mnemonic, 0)
	if err != nil {
		return nil, false, fmt.Errorf("derive from mnemonic: %w", err)
	}
	return &EVMAccount{key: privKey, address: addr}, true, nil
}

// GetWalletAddress returns the primary (EVM) address, or "" if no wallet.
func GetWalletAddress() string {
	w, err := loadWalletData()
	if err != nil || w == nil {
		return ""
	}
	mnemonic := strings.TrimSpace(w.Mnemonic)
	if mnemonic == "" {
		return ""
	}
	_, addr, err := getOrDeriveEVM(mnemonic, getAccountIndex(w))
	if err != nil {
		return ""
	}
	return addr
}

// CurrentIndex returns the current HD account index. Returns 0 if no wallet.
func CurrentIndex() uint32 {
	w, err := loadWalletData()
	if err != nil || w == nil {
		return 0
	}
	return getAccountIndex(w)
}

// DeriveAddress returns the address for the given CAIP-2 chain using the wallet's current index.
// Supports EVM, Morpheum, Bitcoin, Solana via standards.
func DeriveAddress(chainCAIP2 string) (string, error) {
	return DeriveAddressWithIndex(chainCAIP2, 0) // 0 = use wallet's current index
}

// DeriveAddressWithIndex returns the address for the given CAIP-2 chain.
// If index is 0, uses the wallet's current hdAccountIndex.
func DeriveAddressWithIndex(chainCAIP2 string, index uint32) (string, error) {
	w, err := loadWalletData()
	if err != nil {
		return "", err
	}
	if w == nil {
		return "", fmt.Errorf("no wallet: run 'moneyclaw init' first")
	}
	mnemonic := strings.TrimSpace(w.Mnemonic)
	if mnemonic == "" {
		return "", fmt.Errorf("wallet has no mnemonic: run 'moneyclaw init' to create a new wallet")
	}
	if err := validateMnemonic(mnemonic, w.WordCount); err != nil {
		return "", err
	}
	idx := index
	if idx == 0 {
		idx = getAccountIndex(w)
	}
	return DeriveAddressAt(mnemonic, chainCAIP2, idx)
}

// DeriveAddressAt derives the address at an explicit index (internal; uses mnemonic).
func DeriveAddressAt(mnemonic, chainCAIP2 string, index uint32) (string, error) {
	addr, err := getOrDeriveAddress(mnemonic, chainCAIP2, index)
	if err != nil {
		return "", err
	}
	if err := ValidateAddressForChain(addr, chainCAIP2); err != nil {
		return "", fmt.Errorf("validate address: %w", err)
	}
	return addr, nil
}

// DeriveAddressAtExplicitIndex derives the address at the given index (always uses index, never wallet default).
// Used by wallet rotate for preview. Loads wallet internally; does not expose mnemonic.
func DeriveAddressAtExplicitIndex(chainCAIP2 string, index uint32) (string, error) {
	w, err := loadWalletData()
	if err != nil || w == nil {
		return "", fmt.Errorf("no wallet")
	}
	mnemonic := strings.TrimSpace(w.Mnemonic)
	if mnemonic == "" {
		return "", fmt.Errorf("wallet has no mnemonic")
	}
	if err := validateMnemonic(mnemonic, w.WordCount); err != nil {
		return "", err
	}
	return DeriveAddressAt(mnemonic, chainCAIP2, index)
}

// RotateIndex updates the HD account index in wallet.json.
// preview: if true, only shows new addresses without writing.
// Does not sweep funds; operator must migrate manually.
func RotateIndex(newIndex uint32, preview bool) error {
	w, err := loadWalletData()
	if err != nil {
		return err
	}
	if w == nil {
		return fmt.Errorf("no wallet: run 'moneyclaw init' first")
	}
	mnemonic := strings.TrimSpace(w.Mnemonic)
	if mnemonic == "" {
		return fmt.Errorf("wallet has no mnemonic")
	}
	if err := validateMnemonic(mnemonic, w.WordCount); err != nil {
		return err
	}
	oldIdx := getAccountIndex(w)
	if newIndex == oldIdx {
		return fmt.Errorf("index already %d", newIndex)
	}
	if preview {
		return nil // Caller displays addresses
	}
	w.HDAccountIndex = newIndex
	data, _ := json.MarshalIndent(w, "", "  ")
	path := GetWalletPath()
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write wallet: %w", err)
	}
	ClearDerivedKeys()
	return nil
}

// WalletExists returns true if wallet.json exists.
func WalletExists() bool {
	_, err := os.Stat(GetWalletPath())
	return err == nil
}

// WalletMetadata holds wallet info without exposing mnemonic (for API).
type WalletMetadata struct {
	WordCount int    `json:"wordCount"`
	CreatedAt string `json:"createdAt,omitempty"`
}

// GetWalletMetadata returns wallet metadata if wallet exists. Never returns mnemonic.
func GetWalletMetadata() (*WalletMetadata, error) {
	w, err := loadWalletData()
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, nil
	}
	return &WalletMetadata{WordCount: w.WordCount, CreatedAt: w.CreatedAt}, nil
}
