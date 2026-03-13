package inference

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseChatJimmyResponse(t *testing.T) {
	tests := []struct {
		in           string
		wantText     string
		wantIn       int
		wantOut      int
	}{
		{"Hello world<|stats|>{\"prefill_tokens\":42,\"decode_tokens\":10}<|/stats|>", "Hello world", 42, 10},
		{"Hello world", "Hello world", 0, 0},
		{"No stats<|stats|>", "No stats<|stats|>", 0, 0},
	}
	for _, tt := range tests {
		text, in, out := parseChatJimmyResponse(tt.in)
		if text != tt.wantText || in != tt.wantIn || out != tt.wantOut {
			t.Errorf("parseChatJimmyResponse(%q) = %q,%d,%d, want %q,%d,%d", tt.in, text, in, out, tt.wantText, tt.wantIn, tt.wantOut)
		}
	}
}

func TestNewChatJimmyClient_Defaults(t *testing.T) {
	c := NewChatJimmyClient("", "", 0)
	if c.BaseURL != "https://chatjimmy.ai" {
		t.Errorf("BaseURL: got %s", c.BaseURL)
	}
	if c.Model != "llama3.1-8B" {
		t.Errorf("Model: got %s", c.Model)
	}
	if c.MaxTokens != 4096 {
		t.Errorf("MaxTokens: got %d", c.MaxTokens)
	}
}

func TestChatJimmyClient_Chat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("path: got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Hello from ChatJimmy"))
	}))
	defer server.Close()

	c := NewChatJimmyClient(server.URL, "llama3.1-8B", 4096)
	resp, err := c.Chat(context.Background(), []ChatMessage{
		{Role: "user", Content: "Hi"},
	}, nil)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "Hello from ChatJimmy" {
		t.Errorf("Content: got %q", resp.Content)
	}
}

func TestChatJimmyClient_Health(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/health" {
			t.Errorf("path: got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","backend":"healthy"}`))
	}))
	defer server.Close()

	c := NewChatJimmyClient(server.URL, "llama3.1-8B", 4096)
	ok, err := c.Health(context.Background())
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if !ok {
		t.Error("Health: expected true")
	}
}

func TestChatJimmyClient_Health_Unhealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"degraded","backend":"unhealthy"}`))
	}))
	defer server.Close()

	c := NewChatJimmyClient(server.URL, "llama3.1-8B", 4096)
	ok, err := c.Health(context.Background())
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if ok {
		t.Error("Health: expected false for unhealthy")
	}
}

func TestChatJimmyClient_Models(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/models" {
			t.Errorf("path: got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"id":"llama3.1-8B","object":"model","created":0,"owned_by":"Taalas Inc."}]}`))
	}))
	defer server.Close()

	c := NewChatJimmyClient(server.URL, "llama3.1-8B", 4096)
	models, err := c.Models(context.Background())
	if err != nil {
		t.Fatalf("Models: %v", err)
	}
	if len(models) != 1 || models[0] != "llama3.1-8B" {
		t.Errorf("Models: got %v", models)
	}
}

func TestChatJimmyClient_ChatWithStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Hi there<|stats|>{\"prefill_tokens\":5,\"decode_tokens\":2}<|/stats|>"))
	}))
	defer server.Close()

	c := NewChatJimmyClient(server.URL, "llama3.1-8B", 4096)
	resp, err := c.Chat(context.Background(), []ChatMessage{
		{Role: "user", Content: "Hi"},
	}, nil)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "Hi there" {
		t.Errorf("Content: got %q", resp.Content)
	}
	if resp.InputTokens != 5 || resp.OutputTokens != 2 {
		t.Errorf("tokens: got %d/%d, want 5/2", resp.InputTokens, resp.OutputTokens)
	}
}