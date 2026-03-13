package identity

// WalletData is the persisted wallet format (TS-aligned for EVM; extended for multi-chain).
type WalletData struct {
	PrivateKey string `json:"privateKey"`           // 0x-prefixed hex (EVM/secp256k1)
	Mnemonic   string `json:"mnemonic,omitempty"`   // BIP-39 seed for HD derivation (optional)
	CreatedAt  string `json:"createdAt"`
}

// ProvisionResult is returned by Provision (TS-aligned).
type ProvisionResult struct {
	APIKey        string `json:"apiKey"`
	WalletAddress string `json:"walletAddress"`
	KeyPrefix     string `json:"keyPrefix"`
}
