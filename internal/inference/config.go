package inference

import (
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// configKeyDef maps APIKeyConfigKey (PascalCase) to both value getter and JSON key.
// Single source of truth for config lookup and ProviderKeyConfigName.
var apiKeyConfigDefs = map[string]struct {
	jsonKey string
	getter  func(*types.AutomatonConfig) string
}{
	"OpenAIAPIKey":     {"openaiApiKey", func(c *types.AutomatonConfig) string { return c.OpenAIAPIKey }},
	"ConwayAPIKey":     {"conwayApiKey", func(c *types.AutomatonConfig) string { return c.ConwayAPIKey }},
	"GroqAPIKey":       {"groqApiKey", func(c *types.AutomatonConfig) string { return c.GroqAPIKey }},
	"MistralAPIKey":    {"mistralApiKey", func(c *types.AutomatonConfig) string { return c.MistralAPIKey }},
	"DeepSeekAPIKey":   {"deepSeekApiKey", func(c *types.AutomatonConfig) string { return c.DeepSeekAPIKey }},
	"OpenRouterAPIKey": {"openRouterApiKey", func(c *types.AutomatonConfig) string { return c.OpenRouterAPIKey }},
	"XAIAPIKey":        {"xaiApiKey", func(c *types.AutomatonConfig) string { return c.XAIAPIKey }},
	"TogetherAPIKey":   {"togetherApiKey", func(c *types.AutomatonConfig) string { return c.TogetherAPIKey }},
	"FireworksAPIKey":  {"fireworksApiKey", func(c *types.AutomatonConfig) string { return c.FireworksAPIKey }},
	"PerplexityAPIKey": {"perplexityApiKey", func(c *types.AutomatonConfig) string { return c.PerplexityAPIKey }},
	"CohereAPIKey":     {"cohereApiKey", func(c *types.AutomatonConfig) string { return c.CohereAPIKey }},
	"QwenAPIKey":       {"qwenApiKey", func(c *types.AutomatonConfig) string { return c.QwenAPIKey }},
	"MoonshotAPIKey":   {"moonshotApiKey", func(c *types.AutomatonConfig) string { return c.MoonshotAPIKey }},
	"AnthropicAPIKey":  {"anthropicApiKey", func(c *types.AutomatonConfig) string { return c.AnthropicAPIKey }},
	"ZaiAPIKey":        {"zaiApiKey", func(c *types.AutomatonConfig) string { return c.ZaiAPIKey }},
	"GoogleAPIKey":     {"googleApiKey", func(c *types.AutomatonConfig) string { return c.GoogleAPIKey }},
	"HeliconeAPIKey":   {"heliconeApiKey", func(c *types.AutomatonConfig) string { return c.HeliconeAPIKey }},
	"DeepInfraAPIKey":  {"deepInfraApiKey", func(c *types.AutomatonConfig) string { return c.DeepInfraAPIKey }},
	"NovitaAPIKey":     {"novitaApiKey", func(c *types.AutomatonConfig) string { return c.NovitaAPIKey }},
	"SiliconFlowAPIKey": {"siliconFlowApiKey", func(c *types.AutomatonConfig) string { return c.SiliconFlowAPIKey }},
	"CerebrasAPIKey":   {"cerebrasApiKey", func(c *types.AutomatonConfig) string { return c.CerebrasAPIKey }},
	"SambaNovaAPIKey":  {"sambaNovaApiKey", func(c *types.AutomatonConfig) string { return c.SambaNovaAPIKey }},
	"AzureOpenAIAPIKey": {"azureOpenAIApiKey", func(c *types.AutomatonConfig) string { return c.AzureOpenAIAPIKey }},
	"VertexAPIKey":     {"vertexApiKey", func(c *types.AutomatonConfig) string { return c.VertexAPIKey }},
}

var baseURLConfigDefs = map[string]struct {
	jsonKey string
	getter  func(*types.AutomatonConfig) string
}{
	"ConwayAPIURL":         {"conwayApiUrl", func(c *types.AutomatonConfig) string { return c.ConwayAPIURL }},
	"AzureOpenAIEndpoint":  {"azureOpenAIEndpoint", func(c *types.AutomatonConfig) string { return c.AzureOpenAIEndpoint }},
	"VertexAPIURL":         {"vertexApiUrl", func(c *types.AutomatonConfig) string { return c.VertexAPIURL }},
	"OllamaAPIURL":         {"ollamaApiUrl", func(c *types.AutomatonConfig) string { return c.OllamaAPIURL }},
}

// providerResolutionOrder defines backward-compat priority for auto-detecting provider from config.
// First match wins. Special cases: conway/azure/vertex require both key and endpoint.
var providerResolutionOrder = []struct {
	provider     string
	keyKey       string
	baseURLKey   string // required for conway, azure, vertex
}{
	{"openai", "OpenAIAPIKey", ""},
	{"conway", "ConwayAPIKey", "ConwayAPIURL"},
	{"openrouter", "OpenRouterAPIKey", ""},
	{"groq", "GroqAPIKey", ""},
	{"mistral", "MistralAPIKey", ""},
	{"deepseek", "DeepSeekAPIKey", ""},
	{"xai", "XAIAPIKey", ""},
	{"together", "TogetherAPIKey", ""},
	{"fireworks", "FireworksAPIKey", ""},
	{"perplexity", "PerplexityAPIKey", ""},
	{"cohere", "CohereAPIKey", ""},
	{"qwen", "QwenAPIKey", ""},
	{"moonshot", "MoonshotAPIKey", ""},
	{"zai", "ZaiAPIKey", ""},
	{"google", "GoogleAPIKey", ""},
	{"helicone", "HeliconeAPIKey", ""},
	{"deepinfra", "DeepInfraAPIKey", ""},
	{"novita", "NovitaAPIKey", ""},
	{"siliconflow", "SiliconFlowAPIKey", ""},
	{"cerebras", "CerebrasAPIKey", ""},
	{"sambanova", "SambaNovaAPIKey", ""},
	{"azure", "AzureOpenAIAPIKey", "AzureOpenAIEndpoint"},
	{"vertex", "VertexAPIKey", "VertexAPIURL"},
}
