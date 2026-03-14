package signverify

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58"
)

func TestVerifyEthereum_ValidSignature(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	expectedAddr := crypto.PubkeyToAddress(key.PublicKey).Hex()

	message := "Hello, World!"
	hash := createEthereumPersonalMessageHash([]byte(message))
	sig, err := crypto.Sign(hash, key)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if sig[64] < 27 {
		sig[64] += 27
	}
	signature := "0x" + hex.EncodeToString(sig)

	recovered, err := VerifyEthereum(message, signature)
	if err != nil {
		t.Fatalf("VerifyEthereum: %v", err)
	}
	if !strings.EqualFold(recovered, expectedAddr) {
		t.Errorf("recovered %s != expected %s", recovered, expectedAddr)
	}

	ok, err := VerifyEthereumWithAddress(message, signature, expectedAddr)
	if err != nil {
		t.Fatalf("VerifyEthereumWithAddress: %v", err)
	}
	if !ok {
		t.Error("VerifyEthereumWithAddress should return true")
	}
}

func TestVerifyEthereum_InvalidSignature(t *testing.T) {
	message := "Hello, World!"
	signature := "0x" + strings.Repeat("00", 65)
	_, err := VerifyEthereum(message, signature)
	if err == nil {
		t.Error("expected error for invalid signature")
	}
}

func TestVerifyEthereum_InvalidHex(t *testing.T) {
	_, err := VerifyEthereum("hi", "not-hex")
	if err == nil {
		t.Error("expected error for invalid hex")
	}
}

func TestVerifyEthereum_WrongLength(t *testing.T) {
	_, err := VerifyEthereum("hi", "0x1234")
	if err == nil {
		t.Error("expected error for wrong signature length")
	}
}

func TestVerifySolana_ValidSignature(t *testing.T) {
	// Generate Ed25519 keypair, sign, verify
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	address := base58.Encode(pub)
	message := "Hello, Solana!"
	sig := ed25519.Sign(priv, []byte(message))
	signature := base64.StdEncoding.EncodeToString(sig)

	ok, err := VerifySolana(message, signature, address)
	if err != nil {
		t.Fatalf("VerifySolana: %v", err)
	}
	if !ok {
		t.Error("VerifySolana should return true for valid signature")
	}
}
