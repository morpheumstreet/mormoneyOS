package inference

import (
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

func TestNewClientFromConfig_OpenAI(t *testing.T) {
	cfg := &types.AutomatonConfig{
		OpenAIAPIKey:     "sk-test",
		InferenceModel:   "gpt-4o-mini",
		MaxTokensPerTurn: 4096,
	}
	c := NewClientFromConfig(cfg)
	if _, ok := c.(*OpenAICompatibleClient); !ok {
		t.Fatalf("expected OpenAICompatibleClient, got %T", c)
	}
	if c.GetDefaultModel() != "gpt-4o-mini" {
		t.Errorf("model: got %s", c.GetDefaultModel())
	}
}

func TestNewClientFromConfig_Conway(t *testing.T) {
	cfg := &types.AutomatonConfig{
		ConwayAPIURL:     "https://api.conway.tech",
		ConwayAPIKey:     "ck-test",
		InferenceModel:   "gpt-4o",
		MaxTokensPerTurn: 4096,
	}
	c := NewClientFromConfig(cfg)
	if _, ok := c.(*OpenAICompatibleClient); !ok {
		t.Fatalf("expected OpenAICompatibleClient, got %T", c)
	}
}

func TestNewClientFromConfig_ExplicitProvider(t *testing.T) {
	cfg := &types.AutomatonConfig{
		Provider:         "groq",
		GroqAPIKey:       "gsk-test",
		InferenceModel:   "llama-3-70b",
		MaxTokensPerTurn: 4096,
	}
	c := NewClientFromConfig(cfg)
	if _, ok := c.(*OpenAICompatibleClient); !ok {
		t.Fatalf("expected OpenAICompatibleClient, got %T", c)
	}
}

func TestNewClientFromConfig_ChatJimmyWhenNoKeys(t *testing.T) {
	cfg := &types.AutomatonConfig{
		InferenceModel:   "llama3.1-8B",
		MaxTokensPerTurn: 4096,
	}
	c := NewClientFromConfig(cfg)
	// ChatJimmy is the free default when no API keys are configured
	if _, ok := c.(*ChatJimmyClient); !ok {
		t.Fatalf("expected ChatJimmyClient when no keys (free default), got %T", c)
	}
	if c.GetDefaultModel() != "llama3.1-8B" {
		t.Errorf("model: got %s", c.GetDefaultModel())
	}
}

func TestNewClientFromConfig_BackwardCompatPriority(t *testing.T) {
	cfg := &types.AutomatonConfig{
		OpenAIAPIKey:     "sk-openai",
		ConwayAPIURL:     "https://api.conway.tech",
		ConwayAPIKey:     "ck-conway",
		InferenceModel:   "gpt-4o",
		MaxTokensPerTurn: 4096,
	}
	c := NewClientFromConfig(cfg)
	// Should pick OpenAI (higher priority)
	client, ok := c.(*OpenAICompatibleClient)
	if !ok {
		t.Fatalf("expected OpenAICompatibleClient, got %T", c)
	}
	if client.Name != "OpenAI" {
		t.Errorf("expected OpenAI, got %s", client.Name)
	}
}

func TestNewClientFromConfig_ChatJimmyExplicit(t *testing.T) {
	cfg := &types.AutomatonConfig{
		Provider:         "chatjimmy",
		InferenceModel:   "llama3.1-8B",
		MaxTokensPerTurn: 4096,
	}
	c := NewClientFromConfig(cfg)
	if _, ok := c.(*ChatJimmyClient); !ok {
		t.Fatalf("expected ChatJimmyClient, got %T", c)
	}
}

func TestNewClientFromConfig_ChatJimmyAlias(t *testing.T) {
	cfg := &types.AutomatonConfig{
		Provider:         "chatjimmy-cli",
		InferenceModel:   "llama3.1-8B",
		MaxTokensPerTurn: 4096,
	}
	c := NewClientFromConfig(cfg)
	if _, ok := c.(*ChatJimmyClient); !ok {
		t.Fatalf("expected ChatJimmyClient for chatjimmy-cli alias, got %T", c)
	}
}

func TestNewClientFromConfig_ChatJimmyEnvBaseURL(t *testing.T) {
	t.Setenv("CHATJIMMY_BASE_URL", "https://custom.chatjimmy.example")
	defer t.Setenv("CHATJIMMY_BASE_URL", "")

	cfg := &types.AutomatonConfig{
		Provider:         "chatjimmy",
		InferenceModel:   "llama3.1-8B",
		MaxTokensPerTurn: 4096,
	}
	c := NewClientFromConfig(cfg)
	client, ok := c.(*ChatJimmyClient)
	if !ok {
		t.Fatalf("expected ChatJimmyClient, got %T", c)
	}
	if client.BaseURL != "https://custom.chatjimmy.example" {
		t.Errorf("BaseURL: got %s (expected env override)", client.BaseURL)
	}
}

func TestLookupProvider(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"openai", "OpenAI"},
		{"conway", "Conway"},
		{"ollama", "Ollama"},
		{"openrouter", "OpenRouter"},
		{"groq", "Groq"},
		{"xai", "xAI (Grok)"},
		{"unknown", ""},
	}
	for _, tt := range tests {
		spec := LookupProvider(tt.key)
		if tt.want == "" {
			if spec != nil {
				t.Errorf("LookupProvider(%q) = %v, want nil", tt.key, spec)
			}
			continue
		}
		if spec == nil || spec.DisplayName != tt.want {
			t.Errorf("LookupProvider(%q) = %v, want DisplayName %q", tt.key, spec, tt.want)
		}
	}
}

func TestBestEnhanceClient_EmptyModelsUsesChatJimmy(t *testing.T) {
	cfg := &types.AutomatonConfig{MaxTokensPerTurn: 4096}
	c := BestEnhanceClient(cfg)
	if c == nil {
		t.Fatal("BestEnhanceClient returned nil")
	}
	if _, ok := c.(*ChatJimmyClient); !ok {
		t.Errorf("expected ChatJimmyClient when Models empty, got %T", c)
	}
}

func TestBestEnhanceClient_PrefersModelList(t *testing.T) {
	cfg := &types.AutomatonConfig{
		MaxTokensPerTurn: 4096,
		Models: []types.LLMModelEntry{
			{Provider: "openai", ModelID: "gpt-4o-mini", APIKey: "sk-test", Enabled: true, Priority: 0},
		},
	}
	c := BestEnhanceClient(cfg)
	if c == nil {
		t.Fatal("BestEnhanceClient returned nil")
	}
	if _, ok := c.(*OpenAICompatibleClient); !ok {
		t.Errorf("expected OpenAICompatibleClient from Models list, got %T", c)
	}
}

func TestBestEnhanceClient_SkipsDisabledAndMissingKey(t *testing.T) {
	cfg := &types.AutomatonConfig{
		MaxTokensPerTurn: 4096,
		Models: []types.LLMModelEntry{
			{Provider: "openai", ModelID: "gpt-4o", Enabled: false, Priority: 0},
			{Provider: "groq", ModelID: "llama", Enabled: true, Priority: 1}, // no API key
		},
	}
	c := BestEnhanceClient(cfg)
	if c == nil {
		t.Fatal("BestEnhanceClient returned nil")
	}
	// Should fall back to ChatJimmy when no model in list has credentials
	if _, ok := c.(*ChatJimmyClient); !ok {
		t.Errorf("expected ChatJimmyClient fallback when list has no working models, got %T", c)
	}
}
