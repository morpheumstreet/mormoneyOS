package inference

import (
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// ProviderSpec describes an inference provider. OpenAI-compatible providers
// use OpenAICompatibleClient; custom APIs (Anthropic, etc.) use separate impls.
type ProviderSpec struct {
	Key               string
	DisplayName       string
	BaseURL           string
	BaseURLConfigKey  string // when set, override BaseURL from config (e.g. ConwayAPIURL)
	AuthStyle         AuthStyle
	APIKeyConfigKey   string // config field for API key; empty = local (no key required)
	Local             bool
}

var registry = []ProviderSpec{
	// Primary
	{Key: "openai", DisplayName: "OpenAI", BaseURL: "https://api.openai.com", AuthStyle: AuthBearer, APIKeyConfigKey: "OpenAIAPIKey", Local: false},
	{Key: "conway", DisplayName: "Conway", BaseURL: "", BaseURLConfigKey: "ConwayAPIURL", AuthStyle: AuthXApiKey, APIKeyConfigKey: "ConwayAPIKey", Local: false},
	{Key: "ollama", DisplayName: "Ollama", BaseURL: "http://localhost:11434", AuthStyle: AuthBearer, APIKeyConfigKey: "", Local: true},
	// Top models (OpenAI-compatible)
	{Key: "openrouter", DisplayName: "OpenRouter", BaseURL: "https://openrouter.ai/api/v1", AuthStyle: AuthBearer, APIKeyConfigKey: "OpenRouterAPIKey", Local: false},
	{Key: "groq", DisplayName: "Groq", BaseURL: "https://api.groq.com/openai/v1", AuthStyle: AuthBearer, APIKeyConfigKey: "GroqAPIKey", Local: false},
	{Key: "mistral", DisplayName: "Mistral", BaseURL: "https://api.mistral.ai/v1", AuthStyle: AuthBearer, APIKeyConfigKey: "MistralAPIKey", Local: false},
	{Key: "deepseek", DisplayName: "DeepSeek", BaseURL: "https://api.deepseek.com", AuthStyle: AuthBearer, APIKeyConfigKey: "DeepSeekAPIKey", Local: false},
	{Key: "xai", DisplayName: "xAI (Grok)", BaseURL: "https://api.x.ai", AuthStyle: AuthBearer, APIKeyConfigKey: "XAIAPIKey", Local: false},
	{Key: "together", DisplayName: "Together AI", BaseURL: "https://api.together.xyz", AuthStyle: AuthBearer, APIKeyConfigKey: "TogetherAPIKey", Local: false},
	{Key: "fireworks", DisplayName: "Fireworks AI", BaseURL: "https://api.fireworks.ai/inference/v1", AuthStyle: AuthBearer, APIKeyConfigKey: "FireworksAPIKey", Local: false},
	{Key: "perplexity", DisplayName: "Perplexity", BaseURL: "https://api.perplexity.ai", AuthStyle: AuthBearer, APIKeyConfigKey: "PerplexityAPIKey", Local: false},
	{Key: "cohere", DisplayName: "Cohere", BaseURL: "https://api.cohere.com/compatibility", AuthStyle: AuthBearer, APIKeyConfigKey: "CohereAPIKey", Local: false},
	{Key: "qwen", DisplayName: "Qwen (DashScope)", BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", AuthStyle: AuthBearer, APIKeyConfigKey: "QwenAPIKey", Local: false},
	{Key: "moonshot", DisplayName: "Moonshot (Kimi)", BaseURL: "https://api.moonshot.ai/v1", AuthStyle: AuthBearer, APIKeyConfigKey: "MoonshotAPIKey", Local: false},
	{Key: "chatjimmy", DisplayName: "ChatJimmy", BaseURL: "https://chatjimmy.ai", AuthStyle: AuthBearer, APIKeyConfigKey: "", Local: false}, // No auth
}

// LookupProvider returns the spec for the given provider key, or nil if unknown.
func LookupProvider(key string) *ProviderSpec {
	for i := range registry {
		if registry[i].Key == key {
			return &registry[i]
		}
	}
	return nil
}

// ListProviders returns all registered provider specs for the config UI.
func ListProviders() []ProviderSpec {
	out := make([]ProviderSpec, len(registry))
	copy(out, registry)
	return out
}

// getConfigValue returns the API key for the given config key name.
func getConfigValue(cfg *types.AutomatonConfig, key string) string {
	switch key {
	case "OpenAIAPIKey":
		return cfg.OpenAIAPIKey
	case "ConwayAPIKey":
		return cfg.ConwayAPIKey
	case "GroqAPIKey":
		return cfg.GroqAPIKey
	case "MistralAPIKey":
		return cfg.MistralAPIKey
	case "DeepSeekAPIKey":
		return cfg.DeepSeekAPIKey
	case "OpenRouterAPIKey":
		return cfg.OpenRouterAPIKey
	case "XAIAPIKey":
		return cfg.XAIAPIKey
	case "TogetherAPIKey":
		return cfg.TogetherAPIKey
	case "FireworksAPIKey":
		return cfg.FireworksAPIKey
	case "PerplexityAPIKey":
		return cfg.PerplexityAPIKey
	case "CohereAPIKey":
		return cfg.CohereAPIKey
	case "QwenAPIKey":
		return cfg.QwenAPIKey
	case "MoonshotAPIKey":
		return cfg.MoonshotAPIKey
	default:
		return ""
	}
}

// getConfigBaseURL returns the base URL for the given config key name.
func getConfigBaseURL(cfg *types.AutomatonConfig, key string) string {
	switch key {
	case "ConwayAPIURL":
		return cfg.ConwayAPIURL
	default:
		return ""
	}
}

// resolveBaseURL returns the base URL for the provider, from spec or config.
func resolveBaseURL(spec *ProviderSpec, cfg *types.AutomatonConfig) string {
	if spec.BaseURLConfigKey != "" {
		if u := getConfigBaseURL(cfg, spec.BaseURLConfigKey); u != "" {
			return u
		}
	}
	return spec.BaseURL
}
