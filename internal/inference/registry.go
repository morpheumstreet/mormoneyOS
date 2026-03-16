package inference

import (
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// ProviderSpec describes an inference provider. OpenAI-compatible providers
// use OpenAICompatibleClient; custom APIs (Anthropic, etc.) use separate impls.
type ProviderSpec struct {
	Key                  string
	DisplayName          string
	BaseURL              string
	BaseURLConfigKey     string // when set, override BaseURL from config (e.g. ConwayAPIURL)
	ChatCompletionsPath  string // when set, use instead of /v1/chat/completions (e.g. /chat/completions for Z.AI)
	AuthStyle            AuthStyle
	APIKeyConfigKey      string // config field for API key; empty = local (no key required)
	Local                bool
	IsReseller           bool   // true = aggregates/hosts others' models (OpenRouter, Together, Fireworks); false = source model developer
}

var registry = []ProviderSpec{
	// --- Resellers (aggregate/host models from other developers) ---
	{Key: "openrouter", DisplayName: "OpenRouter", BaseURL: "https://openrouter.ai/api/v1", AuthStyle: AuthBearer, APIKeyConfigKey: "OpenRouterAPIKey", Local: false, IsReseller: true},
	{Key: "helicone", DisplayName: "Helicone", BaseURL: "https://ai-gateway.helicone.ai", ChatCompletionsPath: "/chat/completions", AuthStyle: AuthBearer, APIKeyConfigKey: "HeliconeAPIKey", Local: false, IsReseller: true},
	{Key: "together", DisplayName: "Together AI", BaseURL: "https://api.together.xyz", AuthStyle: AuthBearer, APIKeyConfigKey: "TogetherAPIKey", Local: false, IsReseller: true},
	{Key: "fireworks", DisplayName: "Fireworks AI", BaseURL: "https://api.fireworks.ai/inference/v1", AuthStyle: AuthBearer, APIKeyConfigKey: "FireworksAPIKey", Local: false, IsReseller: true},
	{Key: "deepinfra", DisplayName: "DeepInfra", BaseURL: "https://api.deepinfra.com/v1/openai", ChatCompletionsPath: "/chat/completions", AuthStyle: AuthBearer, APIKeyConfigKey: "DeepInfraAPIKey", Local: false, IsReseller: true},
	{Key: "novita", DisplayName: "Novita.ai", BaseURL: "https://api.novita.ai/v1", AuthStyle: AuthBearer, APIKeyConfigKey: "NovitaAPIKey", Local: false, IsReseller: true},
	{Key: "siliconflow", DisplayName: "SiliconFlow", BaseURL: "https://api.siliconflow.cn/v1", AuthStyle: AuthBearer, APIKeyConfigKey: "SiliconFlowAPIKey", Local: false, IsReseller: true},
	{Key: "cerebras", DisplayName: "Cerebras", BaseURL: "https://api.cerebras.ai/v1", AuthStyle: AuthBearer, APIKeyConfigKey: "CerebrasAPIKey", Local: false, IsReseller: true},
	{Key: "sambanova", DisplayName: "SambaNova", BaseURL: "https://api.sambanova.ai/v1", AuthStyle: AuthBearer, APIKeyConfigKey: "SambaNovaAPIKey", Local: false, IsReseller: true},

	// --- Source model developers (original model creators) ---
	{Key: "openai", DisplayName: "OpenAI", BaseURL: "https://api.openai.com", AuthStyle: AuthBearer, APIKeyConfigKey: "OpenAIAPIKey", Local: false},
	{Key: "mistral", DisplayName: "Mistral", BaseURL: "https://api.mistral.ai/v1", AuthStyle: AuthBearer, APIKeyConfigKey: "MistralAPIKey", Local: false},
	{Key: "deepseek", DisplayName: "DeepSeek", BaseURL: "https://api.deepseek.com", AuthStyle: AuthBearer, APIKeyConfigKey: "DeepSeekAPIKey", Local: false},
	{Key: "xai", DisplayName: "xAI (Grok)", BaseURL: "https://api.x.ai", AuthStyle: AuthBearer, APIKeyConfigKey: "XAIAPIKey", Local: false},
	{Key: "qwen", DisplayName: "Qwen (DashScope)", BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", AuthStyle: AuthBearer, APIKeyConfigKey: "QwenAPIKey", Local: false},
	{Key: "moonshot", DisplayName: "Moonshot (Kimi)", BaseURL: "https://api.moonshot.ai/v1", AuthStyle: AuthBearer, APIKeyConfigKey: "MoonshotAPIKey", Local: false},
	{Key: "zai", DisplayName: "Zhipu AI (Z.AI)", BaseURL: "https://api.z.ai/api/paas/v4", ChatCompletionsPath: "/chat/completions", AuthStyle: AuthBearer, APIKeyConfigKey: "ZaiAPIKey", Local: false},
	{Key: "google", DisplayName: "Google (Gemini)", BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai", ChatCompletionsPath: "/chat/completions", AuthStyle: AuthBearer, APIKeyConfigKey: "GoogleAPIKey", Local: false},
	{Key: "groq", DisplayName: "Groq", BaseURL: "https://api.groq.com/openai/v1", AuthStyle: AuthBearer, APIKeyConfigKey: "GroqAPIKey", Local: false},
	{Key: "perplexity", DisplayName: "Perplexity", BaseURL: "https://api.perplexity.ai", AuthStyle: AuthBearer, APIKeyConfigKey: "PerplexityAPIKey", Local: false},
	{Key: "cohere", DisplayName: "Cohere", BaseURL: "https://api.cohere.com/compatibility", AuthStyle: AuthBearer, APIKeyConfigKey: "CohereAPIKey", Local: false},

	// --- Cloud (Azure, Vertex — require endpoint URL in config) ---
	{Key: "azure", DisplayName: "Azure OpenAI", BaseURL: "", BaseURLConfigKey: "AzureOpenAIEndpoint", AuthStyle: AuthBearer, APIKeyConfigKey: "AzureOpenAIAPIKey", Local: false},
	{Key: "vertex", DisplayName: "Google Vertex AI", BaseURL: "", BaseURLConfigKey: "VertexAPIURL", AuthStyle: AuthBearer, APIKeyConfigKey: "VertexAPIKey", Local: false},

	// --- Other (custom, local, free) ---
	{Key: "conway", DisplayName: "Conway", BaseURL: "", BaseURLConfigKey: "ConwayAPIURL", AuthStyle: AuthXApiKey, APIKeyConfigKey: "ConwayAPIKey", Local: false},
	{Key: "ollama", DisplayName: "Ollama", BaseURL: "http://localhost:11434", BaseURLConfigKey: "OllamaAPIURL", AuthStyle: AuthBearer, APIKeyConfigKey: "", Local: true},
	{Key: "localai", DisplayName: "LocalAI", BaseURL: "http://localhost:8080", BaseURLConfigKey: "LocalAIAPIURL", AuthStyle: AuthBearer, APIKeyConfigKey: "", Local: true},
	{Key: "llamacpp", DisplayName: "llama.cpp", BaseURL: "http://localhost:8080", BaseURLConfigKey: "LlamaCppAPIURL", AuthStyle: AuthBearer, APIKeyConfigKey: "", Local: true},
	{Key: "lmstudio", DisplayName: "LM Studio", BaseURL: "http://localhost:1234", BaseURLConfigKey: "LMStudioAPIURL", AuthStyle: AuthBearer, APIKeyConfigKey: "", Local: true},
	{Key: "vllm", DisplayName: "vLLM", BaseURL: "http://localhost:8000", BaseURLConfigKey: "VLLMAPIURL", AuthStyle: AuthBearer, APIKeyConfigKey: "", Local: true},
	{Key: "janai", DisplayName: "Jan AI", BaseURL: "http://localhost:1337", BaseURLConfigKey: "JanAIAPIURL", AuthStyle: AuthBearer, APIKeyConfigKey: "", Local: true},
	{Key: "g4f", DisplayName: "GPT4Free (g4f)", BaseURL: "http://localhost:13145", BaseURLConfigKey: "G4fAPIURL", AuthStyle: AuthBearer, APIKeyConfigKey: "", Local: true},
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

// ProviderKeyConfigName returns the config JSON key for a provider's API key (e.g. "openaiApiKey").
func ProviderKeyConfigName(spec *ProviderSpec) string {
	if spec == nil || spec.APIKeyConfigKey == "" {
		return ""
	}
	if def, ok := apiKeyConfigDefs[spec.APIKeyConfigKey]; ok {
		return def.jsonKey
	}
	return ""
}

// ProvidersWithKeys returns which providers have API keys configured (config or model-level).
// For Azure/Vertex, endpoint URL is also required.
func ProvidersWithKeys(cfg *types.AutomatonConfig, models []types.LLMModelEntry) map[string]bool {
	out := make(map[string]bool)
	for _, p := range registry {
		if p.Local || p.APIKeyConfigKey == "" {
			out[p.Key] = true // Local or no-key providers always "available"
			continue
		}
		hasKey := getConfigValue(cfg, p.APIKeyConfigKey) != ""
		if !hasKey {
			for _, m := range models {
				if m.Provider == p.Key && m.APIKey != "" {
					hasKey = true
					break
				}
			}
		}
		if hasKey && p.BaseURLConfigKey != "" {
			// Azure/Vertex require endpoint URL
			hasKey = getConfigBaseURL(cfg, p.BaseURLConfigKey) != ""
		}
		out[p.Key] = hasKey
	}
	return out
}

// getConfigValue returns the API key for the given config key name.
func getConfigValue(cfg *types.AutomatonConfig, key string) string {
	if def, ok := apiKeyConfigDefs[key]; ok {
		return def.getter(cfg)
	}
	return ""
}

// getConfigBaseURL returns the base URL for the given config key name.
func getConfigBaseURL(cfg *types.AutomatonConfig, key string) string {
	if def, ok := baseURLConfigDefs[key]; ok {
		return def.getter(cfg)
	}
	return ""
}

// GetEndpointValue returns the configured endpoint URL for providers that require it (Azure, Vertex).
func GetEndpointValue(cfg *types.AutomatonConfig, spec *ProviderSpec) string {
	if spec == nil || spec.BaseURLConfigKey == "" {
		return ""
	}
	return getConfigBaseURL(cfg, spec.BaseURLConfigKey)
}

// EndpointConfigJSONKey returns the config JSON key for endpoint/URL override (e.g. "azureOpenAIEndpoint").
func EndpointConfigJSONKey(spec *ProviderSpec) string {
	if spec == nil || spec.BaseURLConfigKey == "" {
		return ""
	}
	if def, ok := baseURLConfigDefs[spec.BaseURLConfigKey]; ok {
		return def.jsonKey
	}
	return ""
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
