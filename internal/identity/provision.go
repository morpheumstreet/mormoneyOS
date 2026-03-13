package identity

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spruceid/siwe-go"
)

const (
	provisionConfigFilename = "config.json"
	defaultProvisionAPIURL  = "https://api.conway.tech"
)

// provisionConfig is the format written by Provision to ~/.automaton/config.json.
type provisionConfig struct {
	APIKey         string `json:"apiKey"`
	WalletAddress  string `json:"walletAddress"`
	ProvisionedAt  string `json:"provisionedAt"`
}

// LoadAPIKeyFromConfig reads apiKey from ~/.automaton/config.json (provision output).
func LoadAPIKeyFromConfig() string {
	path := filepath.Join(GetAutomatonDir(), provisionConfigFilename)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var cfg provisionConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return ""
	}
	return strings.TrimSpace(cfg.APIKey)
}

// Provision runs the SIWE flow: nonce → sign → verify → create API key → save config.
// apiURL defaults to defaultProvisionAPIURL. chainCAIP2 must be EVM (e.g. eip155:8453).
func Provision(apiURL, chainCAIP2 string) (*ProvisionResult, error) {
	if apiURL == "" {
		apiURL = defaultProvisionAPIURL
	}
	apiURL = strings.TrimSuffix(apiURL, "/")
	if !IsEVM(chainCAIP2) {
		return nil, fmt.Errorf("provision requires EVM chain, got %s", chainCAIP2)
	}
	chainID := ChainIDFromCAIP2(chainCAIP2)
	if chainID == 0 {
		return nil, fmt.Errorf("invalid chain %s", chainCAIP2)
	}

	acc, _, err := GetWallet()
	if err != nil {
		return nil, fmt.Errorf("wallet: %w", err)
	}

	// 1. Get nonce
	nonce, err := fetchNonce(apiURL)
	if err != nil {
		return nil, err
	}

	// 2. Build and sign SIWE message
	uri := apiURL + "/v1/auth/verify"
	msg, err := siwe.InitMessage(
		"conway.tech",
		acc.Address(),
		uri,
		nonce,
		map[string]interface{}{
			"statement": "Sign in to Conway as an Automaton to provision an API key.",
			"chainId":   int(chainID),
			"issuedAt": time.Now().UTC().Format(time.RFC3339),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("siwe message: %w", err)
	}
	messageStr := msg.String()
	sig, err := signSIWE([]byte(messageStr), acc.PrivateKey())
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}
	sigHex := "0x" + hex.EncodeToString(sig)

	// 3. Verify → JWT
	accessToken, err := verifySIWE(apiURL, messageStr, sigHex)
	if err != nil {
		return nil, err
	}

	// 4. Create API key
	apiKey, keyPrefix, err := createAPIKey(apiURL, accessToken)
	if err != nil {
		return nil, err
	}

	// 5. Save to config.json
	if err := saveProvisionConfig(apiKey, acc.Address()); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}

	return &ProvisionResult{
		APIKey:        apiKey,
		WalletAddress: acc.Address(),
		KeyPrefix:     keyPrefix,
	}, nil
}

// RegisterParent registers creator with Conway (optional, 404-tolerant).
func RegisterParent(creatorAddress, apiURL, apiKey string) error {
	if apiURL == "" {
		apiURL = defaultProvisionAPIURL
	}
	apiURL = strings.TrimSuffix(apiURL, "/")
	url := apiURL + "/v1/automaton/register-parent"
	body := []byte(fmt.Sprintf(`{"creatorAddress":"%s"}`, creatorAddress))
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("register-parent: %d", resp.StatusCode)
	}
	return nil
}

func fetchNonce(apiURL string) (string, error) {
	url := apiURL + "/v1/auth/nonce"
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("nonce request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("nonce: %d", resp.StatusCode)
	}
	var out struct {
		Nonce string `json:"nonce"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Nonce, nil
}

func signSIWE(message []byte, key *ecdsa.PrivateKey) ([]byte, error) {
	prefix := []byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(message)))
	hash := crypto.Keccak256Hash(append(prefix, message...))
	sig, err := crypto.Sign(hash.Bytes(), key)
	if err != nil {
		return nil, err
	}
	// EIP-155: v must be 27 or 28 for standard format
	sig[64] += 27
	return sig, nil
}

func verifySIWE(apiURL, message, signature string) (string, error) {
	url := apiURL + "/v1/auth/verify"
	body, _ := json.Marshal(map[string]string{"message": message, "signature": signature})
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("verify request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("verify: %d", resp.StatusCode)
	}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.AccessToken, nil
}

func createAPIKey(apiURL, accessToken string) (apiKey, keyPrefix string, err error) {
	url := apiURL + "/v1/auth/api-keys"
	body := []byte(`{"name":"conway-automaton"}`)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("api-keys request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("api-keys: %d", resp.StatusCode)
	}
	var out struct {
		Key      string `json:"key"`
		KeyPrefix string `json:"key_prefix"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", err
	}
	return out.Key, out.KeyPrefix, nil
}

func saveProvisionConfig(apiKey, walletAddress string) error {
	dir := GetAutomatonDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(dir, provisionConfigFilename)
	cfg := provisionConfig{
		APIKey:        apiKey,
		WalletAddress: walletAddress,
		ProvisionedAt: time.Now().UTC().Format(time.RFC3339),
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(path, data, 0600)
}
