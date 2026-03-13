package inference

// ModelSpec describes a known model and its default provider.
// Used for docs, CLI suggestions, and provider-aware defaults.
type ModelSpec struct {
	ModelID     string // API model ID (e.g. gpt-4o, anthropic/claude-sonnet-4)
	Provider    string // default provider key
	DisplayName string
}

// TopModels lists ~30 globally popular models for docs and defaults.
// Order reflects typical usage/rankings; provider must be configured.
var TopModels = []ModelSpec{
	// OpenAI
	{ModelID: "gpt-4o", Provider: "openai", DisplayName: "GPT-4o"},
	{ModelID: "gpt-4o-mini", Provider: "openai", DisplayName: "GPT-4o Mini"},
	{ModelID: "gpt-4-turbo", Provider: "openai", DisplayName: "GPT-4 Turbo"},
	{ModelID: "o1", Provider: "openai", DisplayName: "o1"},
	{ModelID: "o1-mini", Provider: "openai", DisplayName: "o1 Mini"},
	// Anthropic (via OpenRouter or future direct client)
	{ModelID: "anthropic/claude-sonnet-4", Provider: "openrouter", DisplayName: "Claude Sonnet 4"},
	{ModelID: "anthropic/claude-opus-4", Provider: "openrouter", DisplayName: "Claude Opus 4"},
	{ModelID: "anthropic/claude-3-5-haiku", Provider: "openrouter", DisplayName: "Claude 3.5 Haiku"},
	// Google (via OpenRouter)
	{ModelID: "google/gemini-2.5-pro", Provider: "openrouter", DisplayName: "Gemini 2.5 Pro"},
	{ModelID: "google/gemini-2.5-flash", Provider: "openrouter", DisplayName: "Gemini 2.5 Flash"},
	// Meta Llama (Groq, Together, OpenRouter)
	{ModelID: "llama-3.3-70b-versatile", Provider: "groq", DisplayName: "Llama 3.3 70B (Groq)"},
	{ModelID: "meta-llama/Llama-3.3-70B-Instruct-Turbo", Provider: "together", DisplayName: "Llama 3.3 70B (Together)"},
	{ModelID: "meta-llama/llama-3.2-3b-instruct", Provider: "groq", DisplayName: "Llama 3.2 3B"},
	// Mistral
	{ModelID: "mistral-large-latest", Provider: "mistral", DisplayName: "Mistral Large"},
	{ModelID: "mistral-small-latest", Provider: "mistral", DisplayName: "Mistral Small"},
	// DeepSeek
	{ModelID: "deepseek-chat", Provider: "deepseek", DisplayName: "DeepSeek Chat"},
	{ModelID: "deepseek-r1", Provider: "deepseek", DisplayName: "DeepSeek R1"},
	// xAI
	{ModelID: "grok-3", Provider: "xai", DisplayName: "Grok 3"},
	{ModelID: "grok-3-mini", Provider: "xai", DisplayName: "Grok 3 Mini"},
	// Qwen
	{ModelID: "qwen-plus", Provider: "qwen", DisplayName: "Qwen Plus"},
	{ModelID: "qwen-max", Provider: "qwen", DisplayName: "Qwen Max"},
	// Moonshot
	{ModelID: "moonshot-v1-32k", Provider: "moonshot", DisplayName: "Moonshot v1 32K"},
	{ModelID: "moonshot-v1-128k", Provider: "moonshot", DisplayName: "Moonshot v1 128K"},
	// Perplexity
	{ModelID: "sonar", Provider: "perplexity", DisplayName: "Perplexity Sonar"},
	{ModelID: "sonar-pro", Provider: "perplexity", DisplayName: "Perplexity Sonar Pro"},
	// Cohere
	{ModelID: "command-a", Provider: "cohere", DisplayName: "Cohere Command A"},
	// Fireworks
	{ModelID: "accounts/fireworks/models/llama-v3p3-70b-instruct", Provider: "fireworks", DisplayName: "Llama 3.3 70B (Fireworks)"},
	// Ollama (local)
	{ModelID: "llama3.2", Provider: "ollama", DisplayName: "Llama 3.2 (Ollama)"},
	{ModelID: "qwen2.5", Provider: "ollama", DisplayName: "Qwen 2.5 (Ollama)"},
	// ChatJimmy (no auth)
	{ModelID: "llama3.1-8B", Provider: "chatjimmy", DisplayName: "Llama 3.1 8B (ChatJimmy)"},
}

// DefaultModelForProvider returns a suggested model ID for the given provider.
func DefaultModelForProvider(provider string) string {
	for _, m := range TopModels {
		if m.Provider == provider {
			return m.ModelID
		}
	}
	switch provider {
	case "openai":
		return "gpt-4o-mini"
	case "conway":
		return "gpt-4o-mini"
	case "openrouter":
		return "anthropic/claude-sonnet-4"
	case "groq":
		return "llama-3.3-70b-versatile"
	case "mistral":
		return "mistral-large-latest"
	case "deepseek":
		return "deepseek-chat"
	case "xai":
		return "grok-3-mini"
	case "together":
		return "meta-llama/Llama-3.3-70B-Instruct-Turbo"
	case "fireworks":
		return "accounts/fireworks/models/llama-v3p3-70b-instruct"
	case "perplexity":
		return "sonar"
	case "cohere":
		return "command-a"
	case "qwen":
		return "qwen-plus"
	case "moonshot":
		return "moonshot-v1-32k"
	case "ollama":
		return "llama3.2"
	case "chatjimmy":
		return "llama3.1-8B"
	default:
		return "gpt-4o-mini"
	}
}

// ListModelsByProvider returns models for a given provider.
func ListModelsByProvider(provider string) []ModelSpec {
	var out []ModelSpec
	for _, m := range TopModels {
		if m.Provider == provider {
			out = append(out, m)
		}
	}
	return out
}
