package inference

// CatalogEntry describes a model in CanIRun.ai style for the model config UI.
// Used to show a curated list of popular models with specs (VRAM, context, etc.).
type CatalogEntry struct {
	Provider    string   `json:"provider"`
	ModelID     string   `json:"modelId"`
	DisplayName string   `json:"displayName"`
	Params      string   `json:"params"`      // e.g. "8B", "70B"
	VRAMGB     float64  `json:"vramGb"`       // VRAM for local models; 0 for cloud
	ContextK   int      `json:"contextK"`     // context in thousands, e.g. 128
	Arch       string   `json:"arch"`         // "Dense", "MoE"
	UseCases   []string `json:"useCases"`     // chat, code, reasoning, vision
	Tier       string   `json:"tier"`         // S, A, B, C, D, F (local runnability)
	Description string  `json:"description"` // short tagline
}

// ModelCatalog returns a curated list of models in CanIRun.ai / llm-stats.com style.
// Includes cloud API models and popular local (Ollama) models.
func ModelCatalog() []CatalogEntry {
	return []CatalogEntry{
		// --- Cloud API models (VRAM 0) — llm-stats.com inspired ---
		{Provider: "openai", ModelID: "gpt-4o", DisplayName: "GPT-4o", Params: "", VRAMGB: 0, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code", "vision"}, Tier: "S", Description: "OpenAI flagship multimodal"},
		{Provider: "openai", ModelID: "gpt-4o-mini", DisplayName: "GPT-4o Mini", Params: "", VRAMGB: 0, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code"}, Tier: "S", Description: "Fast and capable"},
		{Provider: "openai", ModelID: "o1", DisplayName: "o1", Params: "", VRAMGB: 0, ContextK: 200, Arch: "Dense", UseCases: []string{"reasoning", "code"}, Tier: "S", Description: "OpenAI reasoning model"},
		{Provider: "openai", ModelID: "o1-mini", DisplayName: "o1 Mini", Params: "", VRAMGB: 0, ContextK: 128, Arch: "Dense", UseCases: []string{"reasoning", "code"}, Tier: "S", Description: "Efficient reasoning"},
		{Provider: "openrouter", ModelID: "anthropic/claude-sonnet-4", DisplayName: "Claude Sonnet 4", Params: "", VRAMGB: 0, ContextK: 200, Arch: "Dense", UseCases: []string{"chat", "code", "reasoning"}, Tier: "S", Description: "Anthropic flagship"},
		{Provider: "openrouter", ModelID: "anthropic/claude-opus-4", DisplayName: "Claude Opus 4", Params: "", VRAMGB: 0, ContextK: 200, Arch: "Dense", UseCases: []string{"chat", "code", "reasoning"}, Tier: "S", Description: "Most capable Claude"},
		{Provider: "google", ModelID: "gemini-2.5-pro", DisplayName: "Gemini 2.5 Pro", Params: "", VRAMGB: 0, ContextK: 1000, Arch: "Dense", UseCases: []string{"chat", "code", "vision"}, Tier: "S", Description: "Google flagship (direct)"},
		{Provider: "google", ModelID: "gemini-2.5-flash", DisplayName: "Gemini 2.5 Flash", Params: "", VRAMGB: 0, ContextK: 1000, Arch: "Dense", UseCases: []string{"chat", "code", "vision"}, Tier: "S", Description: "Google fast flagship"},
		{Provider: "groq", ModelID: "llama-3.3-70b-versatile", DisplayName: "Llama 3.3 70B", Params: "70B", VRAMGB: 0, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code", "reasoning"}, Tier: "S", Description: "Best open model at 70B (Groq)"},
		{Provider: "groq", ModelID: "meta-llama/llama-3.2-3b-instruct", DisplayName: "Llama 3.2 3B", Params: "3B", VRAMGB: 0, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code"}, Tier: "A", Description: "Lightweight Llama"},
		{Provider: "deepseek", ModelID: "deepseek-chat", DisplayName: "DeepSeek Chat", Params: "", VRAMGB: 0, ContextK: 64, Arch: "Dense", UseCases: []string{"chat", "code"}, Tier: "S", Description: "Strong coding model"},
		{Provider: "deepseek", ModelID: "deepseek-r1", DisplayName: "DeepSeek R1", Params: "", VRAMGB: 0, ContextK: 64, Arch: "MoE", UseCases: []string{"reasoning", "code"}, Tier: "S", Description: "Reasoning-focused"},
		{Provider: "mistral", ModelID: "mistral-large-latest", DisplayName: "Mistral Large", Params: "", VRAMGB: 0, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code"}, Tier: "S", Description: "Mistral flagship"},
		{Provider: "mistral", ModelID: "mistral-small-latest", DisplayName: "Mistral Small", Params: "", VRAMGB: 0, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code"}, Tier: "A", Description: "Efficient Mistral"},
		{Provider: "xai", ModelID: "grok-3", DisplayName: "Grok 3", Params: "", VRAMGB: 0, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code"}, Tier: "S", Description: "xAI flagship"},
		{Provider: "xai", ModelID: "grok-3-mini", DisplayName: "Grok 3 Mini", Params: "", VRAMGB: 0, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code"}, Tier: "A", Description: "Efficient Grok"},
		{Provider: "together", ModelID: "meta-llama/Llama-3.3-70B-Instruct-Turbo", DisplayName: "Llama 3.3 70B (Together)", Params: "70B", VRAMGB: 0, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code", "reasoning"}, Tier: "S", Description: "Llama 3.3 via Together"},
		{Provider: "fireworks", ModelID: "accounts/fireworks/models/llama-v3p3-70b-instruct", DisplayName: "Llama 3.3 70B (Fireworks)", Params: "70B", VRAMGB: 0, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code"}, Tier: "S", Description: "Llama 3.3 via Fireworks"},
		{Provider: "chatjimmy", ModelID: "llama3.1-8B", DisplayName: "Llama 3.1 8B (ChatJimmy)", Params: "8B", VRAMGB: 0, ContextK: 8, Arch: "Dense", UseCases: []string{"chat", "code"}, Tier: "A", Description: "Free, no API key"},
		{Provider: "google", ModelID: "gemini-2.0-flash", DisplayName: "Gemini 2.0 Flash", Params: "", VRAMGB: 0, ContextK: 1000, Arch: "Dense", UseCases: []string{"chat", "code", "vision"}, Tier: "S", Description: "Google 2.0 Flash"},
		{Provider: "openrouter", ModelID: "google/gemini-2.5-pro", DisplayName: "Gemini 2.5 Pro (via OpenRouter)", Params: "", VRAMGB: 0, ContextK: 1000, Arch: "Dense", UseCases: []string{"chat", "code", "vision"}, Tier: "S", Description: "Same model via OpenRouter"},
		{Provider: "helicone", ModelID: "gpt-4o-mini", DisplayName: "GPT-4o Mini (Helicone)", Params: "", VRAMGB: 0, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code"}, Tier: "S", Description: "100+ models, unified API"},
		{Provider: "deepinfra", ModelID: "meta-llama/Meta-Llama-3-8B-Instruct", DisplayName: "Llama 3 8B (DeepInfra)", Params: "8B", VRAMGB: 0, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code"}, Tier: "S", Description: "Cost-effective inference"},
		{Provider: "cerebras", ModelID: "gpt-oss-120b", DisplayName: "GPT OSS 120B (Cerebras)", Params: "120B", VRAMGB: 0, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code", "reasoning"}, Tier: "S", Description: "World's fastest inference"},
		{Provider: "novita", ModelID: "gpt-4o-mini", DisplayName: "GPT-4o Mini (Novita)", Params: "", VRAMGB: 0, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code"}, Tier: "S", Description: "Low-cost inference platform"},
		{Provider: "siliconflow", ModelID: "Qwen/Qwen2.5-7B-Instruct", DisplayName: "Qwen 2.5 7B (SiliconFlow)", Params: "7B", VRAMGB: 0, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code"}, Tier: "S", Description: "Chinese inference, DeepSeek support"},
		{Provider: "sambanova", ModelID: "Meta-Llama-3.1-70B-Instruct", DisplayName: "Llama 3.1 70B (SambaNova)", Params: "70B", VRAMGB: 0, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code"}, Tier: "S", Description: "Enterprise inference"},
		{Provider: "qwen", ModelID: "qwen-plus", DisplayName: "Qwen Plus", Params: "", VRAMGB: 0, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code", "multilingual"}, Tier: "S", Description: "Alibaba Qwen flagship"},
		{Provider: "qwen", ModelID: "qwen-turbo", DisplayName: "Qwen Turbo", Params: "", VRAMGB: 0, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code"}, Tier: "A", Description: "Fast Qwen model"},
		{Provider: "moonshot", ModelID: "moonshot-v1-8k", DisplayName: "Kimi Moonshot", Params: "", VRAMGB: 0, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code"}, Tier: "S", Description: "Moonshot AI Kimi"},
		{Provider: "zai", ModelID: "glm-5", DisplayName: "GLM-5", Params: "744B", VRAMGB: 0, ContextK: 200, Arch: "MoE", UseCases: []string{"chat", "code", "reasoning"}, Tier: "S", Description: "Zhipu AI flagship, SOTA open-weight"},
		{Provider: "zai", ModelID: "glm-4.6", DisplayName: "GLM-4.6", Params: "", VRAMGB: 0, ContextK: 131, Arch: "Dense", UseCases: []string{"chat", "code"}, Tier: "S", Description: "Zhipu AI GLM-4.6"},

		// --- Local (Ollama) models — CanIRun.ai style ---
		{Provider: "ollama", ModelID: "llama3.2", DisplayName: "Llama 3.2", Params: "3B", VRAMGB: 1.5, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code"}, Tier: "S", Description: "Lightweight Llama for edge"},
		{Provider: "ollama", ModelID: "llama3.1", DisplayName: "Llama 3.1 8B", Params: "8B", VRAMGB: 4.1, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code", "reasoning"}, Tier: "S", Description: "Great quality/speed ratio"},
		{Provider: "ollama", ModelID: "qwen2.5", DisplayName: "Qwen 2.5 7B", Params: "7B", VRAMGB: 3.6, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code", "multilingual"}, Tier: "S", Description: "Strong multilingual and coding"},
		{Provider: "ollama", ModelID: "qwen2.5:7b", DisplayName: "Qwen 2.5 7B (explicit)", Params: "7B", VRAMGB: 3.6, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code"}, Tier: "S", Description: "Qwen 2.5 7B"},
		{Provider: "ollama", ModelID: "qwen2.5-coder:7b", DisplayName: "Qwen 2.5 Coder 7B", Params: "7B", VRAMGB: 3.6, ContextK: 128, Arch: "Dense", UseCases: []string{"code"}, Tier: "S", Description: "Dedicated coding model"},
		{Provider: "ollama", ModelID: "phi3", DisplayName: "Phi-3.5 Mini", Params: "3.8B", VRAMGB: 1.9, ContextK: 128, Arch: "Dense", UseCases: []string{"reasoning", "code", "chat"}, Tier: "S", Description: "Efficient small model"},
		{Provider: "ollama", ModelID: "phi4", DisplayName: "Phi-4 14B", Params: "14B", VRAMGB: 7.2, ContextK: 16, Arch: "Dense", UseCases: []string{"reasoning", "code"}, Tier: "A", Description: "Microsoft reasoning-focused"},
		{Provider: "ollama", ModelID: "mistral", DisplayName: "Mistral 7B", Params: "7B", VRAMGB: 3.6, ContextK: 32, Arch: "Dense", UseCases: []string{"chat", "reasoning"}, Tier: "S", Description: "High-quality 7B"},
		{Provider: "ollama", ModelID: "gemma2", DisplayName: "Gemma 2 9B", Params: "9B", VRAMGB: 4.6, ContextK: 8, Arch: "Dense", UseCases: []string{"chat", "reasoning"}, Tier: "A", Description: "Google mid-size open model"},
		{Provider: "ollama", ModelID: "deepseek-r1:7b", DisplayName: "DeepSeek R1 Distill 7B", Params: "7B", VRAMGB: 3.6, ContextK: 64, Arch: "Dense", UseCases: []string{"reasoning"}, Tier: "S", Description: "R1 reasoning distilled"},
		{Provider: "ollama", ModelID: "qwen2.5:14b", DisplayName: "Qwen 2.5 14B", Params: "14B", VRAMGB: 7.2, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code", "reasoning"}, Tier: "A", Description: "Excellent for size class"},
		{Provider: "ollama", ModelID: "qwen2.5-coder:32b", DisplayName: "Qwen 2.5 Coder 32B", Params: "32B", VRAMGB: 16.4, ContextK: 128, Arch: "Dense", UseCases: []string{"code"}, Tier: "B", Description: "Best open-source coding at release"},
		{Provider: "ollama", ModelID: "llama3.1:70b", DisplayName: "Llama 3.1 70B", Params: "70B", VRAMGB: 35.9, ContextK: 128, Arch: "Dense", UseCases: []string{"chat", "code", "reasoning"}, Tier: "C", Description: "Best open model at 70B"},
		{Provider: "ollama", ModelID: "mixtral", DisplayName: "Mixtral 8x7B", Params: "47B", VRAMGB: 24.1, ContextK: 32, Arch: "MoE", UseCases: []string{"chat", "code"}, Tier: "B", Description: "MoE with 12.9B active"},
	}
}
