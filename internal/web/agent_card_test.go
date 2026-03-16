package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockAgentCardDB struct {
	kv map[string]string
}

func (m *mockAgentCardDB) GetKV(key string) (string, bool, error) {
	v, ok := m.kv[key]
	return v, ok, nil
}

func (m *mockAgentCardDB) SetKV(key, value string) error {
	if m.kv == nil {
		m.kv = make(map[string]string)
	}
	m.kv[key] = value
	return nil
}

func (m *mockAgentCardDB) GetIdentity(key string) (string, bool, error) {
	v, ok := m.kv[key]
	return v, ok, nil
}

func (m *mockAgentCardDB) SetAgentState(state string) error   { return nil }
func (m *mockAgentCardDB) GetAgentState() (string, bool, error) { return "", false, nil }
func (m *mockAgentCardDB) DeleteKV(key string) error           { return nil }
func (m *mockAgentCardDB) InsertWakeEvent(source, reason string) error { return nil }
func (m *mockAgentCardDB) GetTurnCount() (int64, error)        { return 0, nil }

func TestHandleWellKnownAgentCard(t *testing.T) {
	db := &mockAgentCardDB{kv: map[string]string{
		"address":           "0x1234567890abcdef1234567890abcdef12345678",
		"address_eip155:8453": "0x1234567890abcdef1234567890abcdef12345678",
		"soul_content": `## Core Purpose
I am a financial optimization agent.

## Capabilities
- Budget tracking
- Portfolio analysis`,
	}}
	cfg := &ServerConfig{
		Name:           "TestAgent",
		WalletAddress:  "0x1234567890abcdef1234567890abcdef12345678",
		CreatorAddress: "0xcreator",
	}
	srv := NewServer(":0", &RuntimeState{}, db, cfg, nil)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/agent-card.json", nil)
	req.Host = "agent.example.com"
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/ld+json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want application/ld+json", ct)
	}

	var card map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&card); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if card["@type"] != "SoftwareApplication" {
		t.Errorf("@type = %v", card["@type"])
	}
	if card["name"] != "TestAgent" {
		t.Errorf("name = %v", card["name"])
	}
	// In tests r.TLS is nil, so scheme is http
	if card["url"] != "http://agent.example.com/.well-known/agent-card.json" {
		t.Errorf("url = %v", card["url"])
	}
	desc, _ := card["description"].(string)
	if desc == "" || len(desc) < 10 {
		t.Errorf("description empty or too short: %q", desc)
	}
	ids, ok := card["identifier"].([]any)
	if !ok || len(ids) == 0 {
		t.Error("identifier missing or empty")
	}
}

func TestHandleWellKnownAgentCard_MethodNotAllowed(t *testing.T) {
	srv := NewServer(":0", &RuntimeState{}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/.well-known/agent-card.json", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST status = %d, want 405", rec.Code)
	}
}
