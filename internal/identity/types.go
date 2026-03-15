package identity

// WalletData is the persisted wallet format. Mnemonic-only; single source of truth.
// No private keys stored; keys derived on demand via standards MultiChainKeyManager.
type WalletData struct {
	Mnemonic       string `json:"mnemonic"`                 // BIP-39 phrase; only persisted secret
	CreatedAt      string `json:"createdAt"`               // ISO 8601
	HDAccountIndex uint32 `json:"hdAccountIndex,omitempty"` // Derivation index; default 0
	WordCount      int    `json:"wordCount,omitempty"`     // 12|15|18|21|24 for validation
}

// ProvisionResult is returned by Provision (TS-aligned).
type ProvisionResult struct {
	APIKey        string `json:"apiKey"`
	WalletAddress string `json:"walletAddress"`
	KeyPrefix     string `json:"keyPrefix"`
}
