package inference

import (
	"os"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// NewClientFromConfig returns a real inference client when API keys are configured,
// otherwise a stub. Uses provider registry when provider is set; else backward-compat auto-detect.
func NewClientFromConfig(cfg *types.AutomatonConfig) Client {
	maxTokens := cfg.MaxTokensPerTurn
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	provider := cfg.Provider
	if provider == "" {
		provider = ResolveProviderFromConfig(cfg)
	}
	// Normalize alias
	if provider == "chatjimmy-cli" {
		provider = "chatjimmy"
	}

	model := cfg.InferenceModel
	if model == "" {
		model = DefaultModelForProvider(provider)
		if model == "" {
			model = "gpt-4o-mini"
		}
	}

	// ChatJimmy: custom API, no auth (align with chatjimmy-cli: config > env > default)
	if provider == "chatjimmy" {
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

	spec := LookupProvider(provider)
	if spec == nil {
		return NewStubClient(model)
	}

	key := getConfigValue(cfg, spec.APIKeyConfigKey)
	if key == "" && !spec.Local {
		return NewStubClient(model)
	}

	baseURL := resolveBaseURL(spec, cfg)
	return NewOpenAICompatibleClient(spec.DisplayName, baseURL, key, spec.AuthStyle, model, maxTokens)
}

// ResolveProviderFromConfig returns provider key from config keys (backward compat).
// Priority: OpenAI > Conway > ... > ChatJimmy (fallback when no keys).
func ResolveProviderFromConfig(cfg *types.AutomatonConfig) string {
	return resolveProviderFromKeys(cfg)
}

// resolveProviderFromKeys returns provider key from config keys (backward compat).
func resolveProviderFromKeys(cfg *types.AutomatonConfig) string {
	if cfg.OpenAIAPIKey != "" {
		return "openai"
	}
	if cfg.ConwayAPIURL != "" && cfg.ConwayAPIKey != "" {
		return "conway"
	}
	if cfg.OpenRouterAPIKey != "" {
		return "openrouter"
	}
	if cfg.GroqAPIKey != "" {
		return "groq"
	}
	if cfg.MistralAPIKey != "" {
		return "mistral"
	}
	if cfg.DeepSeekAPIKey != "" {
		return "deepseek"
	}
	if cfg.XAIAPIKey != "" {
		return "xai"
	}
	if cfg.TogetherAPIKey != "" {
		return "together"
	}
	if cfg.FireworksAPIKey != "" {
		return "fireworks"
	}
	if cfg.PerplexityAPIKey != "" {
		return "perplexity"
	}
	if cfg.CohereAPIKey != "" {
		return "cohere"
	}
	if cfg.QwenAPIKey != "" {
		return "qwen"
	}
	if cfg.MoonshotAPIKey != "" {
		return "moonshot"
	}
	// ChatJimmy: no auth, free default when no keys
	return "chatjimmy"
}
