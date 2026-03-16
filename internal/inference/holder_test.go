package inference

import (
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

func TestInferenceClientHolder_Reload(t *testing.T) {
	cfg1 := &types.AutomatonConfig{
		Provider:         "chatjimmy",
		InferenceModel:   "llama3.1-8B",
		MaxTokensPerTurn: 4096,
	}
	cfg2 := &types.AutomatonConfig{
		Provider:         "ollama",
		InferenceModel:   "llama3.1",
		OllamaAPIURL:     "http://localhost:11434",
		MaxTokensPerTurn: 4096,
	}

	h := NewInferenceClientHolder(cfg1)
	c := h.Client()
	if c == nil {
		t.Fatal("Client() returned nil")
	}
	if c.GetDefaultModel() != "llama3.1-8B" {
		t.Errorf("initial model: got %s", c.GetDefaultModel())
	}

	h.Reload(cfg2)
	c = h.Client()
	if c == nil {
		t.Fatal("Client() returned nil after reload")
	}
	if c.GetDefaultModel() != "llama3.1" {
		t.Errorf("after reload model: got %s", c.GetDefaultModel())
	}
}

func TestLiveInferenceClient_Delegates(t *testing.T) {
	cfg := &types.AutomatonConfig{
		Provider:         "chatjimmy",
		InferenceModel:   "llama3.1-8B",
		MaxTokensPerTurn: 4096,
	}
	h := NewInferenceClientHolder(cfg)
	live := h.LiveClient()
	if live == nil {
		t.Fatal("LiveClient() returned nil")
	}
	if live.GetDefaultModel() != "llama3.1-8B" {
		t.Errorf("LiveClient model: got %s", live.GetDefaultModel())
	}

	// Reload and verify live client picks up new client
	cfg2 := &types.AutomatonConfig{
		Provider:         "ollama",
		InferenceModel:   "qwen2.5",
		OllamaAPIURL:     "http://localhost:11434",
		MaxTokensPerTurn: 4096,
	}
	h.Reload(cfg2)
	if live.GetDefaultModel() != "qwen2.5" {
		t.Errorf("LiveClient after reload: got %s", live.GetDefaultModel())
	}
}
