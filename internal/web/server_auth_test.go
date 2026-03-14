package web

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestHandleAPIAuthVerify_Ethereum(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey).Hex()
	message := "Hello, World!"
	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(message))
	hash := crypto.Keccak256Hash(append([]byte(prefix), message...)).Bytes()
	sig, err := crypto.Sign(hash, key)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if sig[64] < 27 {
		sig[64] += 27
	}
	signature := "0x" + hex.EncodeToString(sig)

	srv := NewServer(":0", &RuntimeState{}, nil, nil, nil)
	body, _ := json.Marshal(map[string]string{
		"chain":     "ethereum",
		"message":   message,
		"signature": signature,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Valid   bool   `json:"valid"`
		Address string `json:"address"`
		Token   string `json:"token"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Valid {
		t.Error("expected valid=true")
	}
	if resp.Address != addr {
		t.Errorf("address %s != expected %s", resp.Address, addr)
	}
	if resp.Token == "" {
		t.Error("expected token to be issued for user operations")
	}
}

func TestHandleAPIAuthVerify_CreatorMismatch(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	_ = crypto.PubkeyToAddress(key.PublicKey).Hex() // signer address (different from creator)
	message := "Hello, World!"
	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(message))
	hash := crypto.Keccak256Hash(append([]byte(prefix), message...)).Bytes()
	sig, err := crypto.Sign(hash, key)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if sig[64] < 27 {
		sig[64] += 27
	}
	signature := "0x" + hex.EncodeToString(sig)

	// CreatorAddress set to a different address - login should be rejected
	srv := NewServer(":0", &RuntimeState{}, nil, &ServerConfig{
		JWTSecret:      "test-secret",
		CreatorAddress: "0x0000000000000000000000000000000000000001",
	}, nil)
	body, _ := json.Marshal(map[string]string{
		"chain":     "ethereum",
		"message":   message,
		"signature": signature,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Valid bool   `json:"valid"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Valid {
		t.Error("expected valid=false when address does not match creator")
	}
	if resp.Error != "wallet address does not match creator" {
		t.Errorf("unexpected error: %s", resp.Error)
	}
}

func TestHandleAPIAuthVerify_CreatorMatch(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey).Hex()
	message := "Hello, World!"
	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(message))
	hash := crypto.Keccak256Hash(append([]byte(prefix), message...)).Bytes()
	sig, err := crypto.Sign(hash, key)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if sig[64] < 27 {
		sig[64] += 27
	}
	signature := "0x" + hex.EncodeToString(sig)

	srv := NewServer(":0", &RuntimeState{}, nil, &ServerConfig{
		JWTSecret:      "test-secret",
		CreatorAddress: addr, // matches signer
	}, nil)
	body, _ := json.Marshal(map[string]string{
		"chain":     "ethereum",
		"message":   message,
		"signature": signature,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Valid   bool   `json:"valid"`
		Address string `json:"address"`
		Token   string `json:"token"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Valid || resp.Token == "" {
		t.Error("expected valid=true and token when address matches creator")
	}
}
