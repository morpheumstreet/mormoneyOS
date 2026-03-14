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
