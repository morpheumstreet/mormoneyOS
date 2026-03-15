package inference

import (
	"os"
	"sort"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// normalizeProvider maps provider aliases to canonical keys.
func normalizeProvider(provider string) string {
	if provider == "chatjimmy-cli" {
		return "chatjimmy"
	}
	return provider
}

// newChatJimmyClientFromConfig creates a ChatJimmy client from config (config > env > default).
func newChatJimmyClientFromConfig(cfg *types.AutomatonConfig, model string, maxTokens int) *ChatJimmyClient {
	baseURL := cfg.ChatJimmyAPIURL
	if baseURL == "" {
		if v := os.Getenv("CHATJIMMY_BASE_URL"); v != "" {
			baseURL = v
		}
	}
	if baseURL == "" {
		baseURL = "https://chatjimmy.ai"
	}
	return NewChatJimmyClient(baseURL, model, maxTokens)
}

// NewClientFromConfig returns a real inference client when API keys are configured,
// otherwise a stub. Uses provider registry when provider is set; else backward-compat auto-detect.
func NewClientFromConfig(cfg *types.AutomatonConfig) Client {
	maxTokens := cfg.MaxTokensPerTurn
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	provider := normalizeProvider(cfg.Provider)
	if provider == "" {
		provider = ResolveProviderFromConfig(cfg)
	}

	model := cfg.InferenceModel
	if model == "" {
		model = DefaultModelForProvider(provider)
		if model == "" {
			model = "gpt-4o-mini"
		}
	}

	if provider == "chatjimmy" {
		return newChatJimmyClientFromConfig(cfg, model, maxTokens)
	}

	spec := LookupProvider(provider)
	if spec == nil {
		return NewStubClient(model)
	}

	key := getConfigValue(cfg, spec.APIKeyConfigKey)
	if key == "" && !spec.Local {
		return NewStubClient(model)
	}

	baseURL := resolveBaseURL(spec, cfg)
	if spec.ChatCompletionsPath != "" {
		return NewOpenAICompatibleClientWithPath(spec.DisplayName, baseURL, spec.ChatCompletionsPath, key, spec.AuthStyle, model, maxTokens)
	}
	return NewOpenAICompatibleClient(spec.DisplayName, baseURL, key, spec.AuthStyle, model, maxTokens)
}

// ResolveProviderFromConfig returns provider key from config keys (backward compat).
// Priority: OpenAI > Conway > ... > ChatJimmy (fallback when no keys).
func ResolveProviderFromConfig(cfg *types.AutomatonConfig) string {
	return resolveProviderFromKeys(cfg)
}

// NewClientForModelEntry creates an inference client for a specific LLMModelEntry.
// Uses entry.APIKey if set, else the config's provider API key. Returns nil if provider
// requires a key and none is available.
func NewClientForModelEntry(cfg *types.AutomatonConfig, entry *types.LLMModelEntry) Client {
	if entry == nil || !entry.Enabled {
		return nil
	}
	provider := normalizeProvider(entry.Provider)
	if provider == "" {
		provider = "chatjimmy"
	}
	model := entry.ModelID
	if model == "" {
		model = DefaultModelForProvider(provider)
	}
	maxTokens := cfg.MaxTokensPerTurn
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	if provider == "chatjimmy" {
		return newChatJimmyClientFromConfig(cfg, model, maxTokens)
	}

	spec := LookupProvider(provider)
	if spec == nil {
		return nil
	}
	apiKey := entry.APIKey
	if apiKey == "" {
		apiKey = getConfigValue(cfg, spec.APIKeyConfigKey)
	}
	if apiKey == "" && !spec.Local {
		return nil
	}
	baseURL := resolveBaseURL(spec, cfg)
	if spec.ChatCompletionsPath != "" {
		return NewOpenAICompatibleClientWithPath(spec.DisplayName, baseURL, spec.ChatCompletionsPath, apiKey, spec.AuthStyle, model, maxTokens)
	}
	return NewOpenAICompatibleClient(spec.DisplayName, baseURL, apiKey, spec.AuthStyle, model, maxTokens)
}

// BestEnhanceClient returns the best available client for soul enhancement:
// first enabled model from cfg.Models (by priority), or ChatJimmy as fallback.
func BestEnhanceClient(cfg *types.AutomatonConfig) Client {
	// Sort models by priority (lower = first)
	models := make([]types.LLMModelEntry, len(cfg.Models))
	copy(models, cfg.Models)
	sort.Slice(models, func(i, j int) bool { return models[i].Priority < models[j].Priority })

	for i := range models {
		client := NewClientForModelEntry(cfg, &models[i])
		if client != nil {
			return client
		}
	}
	// Fallback: ChatJimmy (no auth, always available when no better model in list)
	return NewClientForModelEntry(cfg, &types.LLMModelEntry{
		Provider: "chatjimmy",
		ModelID:  DefaultModelForProvider("chatjimmy"),
		Enabled:  true,
	})
}

// resolveProviderFromKeys returns provider key from config keys (backward compat).
// Driven by providerResolutionOrder; first match wins.
func resolveProviderFromKeys(cfg *types.AutomatonConfig) string {
	for _, r := range providerResolutionOrder {
		hasKey := getConfigValue(cfg, r.keyKey) != ""
		if r.baseURLKey != "" {
			hasKey = hasKey && getConfigBaseURL(cfg, r.baseURLKey) != ""
		}
		if hasKey {
			return r.provider
		}
	}
	return "chatjimmy"
}
