package types

// AgentState represents the runtime state of the automaton.
type AgentState string

const (
	AgentStateSetup       AgentState = "setup"
	AgentStateWaking      AgentState = "waking"
	AgentStateRunning     AgentState = "running"
	AgentStateSleeping    AgentState = "sleeping"
	AgentStateLowCompute  AgentState = "low_compute"
	AgentStateCritical    AgentState = "critical"
	AgentStateDead        AgentState = "dead"
)

// SurvivalTier represents the agent's financial survival tier.
type SurvivalTier string

const (
	SurvivalTierHigh       SurvivalTier = "high"
	SurvivalTierNormal     SurvivalTier = "normal"
	SurvivalTierLowCompute SurvivalTier = "low_compute"
	SurvivalTierCritical   SurvivalTier = "critical"
	SurvivalTierDead       SurvivalTier = "dead"
)

// RiskLevel for tool execution policy.
type RiskLevel string

const (
	RiskSafe     RiskLevel = "safe"
	RiskCaution  RiskLevel = "caution"
	RiskDangerous RiskLevel = "dangerous"
	RiskForbidden RiskLevel = "forbidden"
)

// AutomatonConfig is the main configuration.
type AutomatonConfig struct {
	Name               string         `json:"name"`
	GenesisPrompt      string         `json:"genesisPrompt"`
	CreatorMessage     string         `json:"creatorMessage,omitempty"`
	CreatorAddress     string         `json:"creatorAddress"`
	SandboxID          string         `json:"sandboxId"`
	Provider           string         `json:"provider,omitempty"` // "openai", "conway", "ollama", "groq", etc.
	ConwayAPIURL       string         `json:"conwayApiUrl"`
	ConwayAPIKey       string         `json:"conwayApiKey,omitempty"`
	OpenAIAPIKey       string         `json:"openaiApiKey,omitempty"`
	AnthropicAPIKey    string         `json:"anthropicApiKey,omitempty"`
	GroqAPIKey         string         `json:"groqApiKey,omitempty"`
	MistralAPIKey      string         `json:"mistralApiKey,omitempty"`
	DeepSeekAPIKey     string         `json:"deepSeekApiKey,omitempty"`
	OpenRouterAPIKey   string         `json:"openRouterApiKey,omitempty"`
	XAIAPIKey          string         `json:"xaiApiKey,omitempty"`
	TogetherAPIKey     string         `json:"togetherApiKey,omitempty"`
	FireworksAPIKey    string         `json:"fireworksApiKey,omitempty"`
	PerplexityAPIKey   string         `json:"perplexityApiKey,omitempty"`
	CohereAPIKey       string         `json:"cohereApiKey,omitempty"`
	QwenAPIKey         string         `json:"qwenApiKey,omitempty"`
	MoonshotAPIKey     string         `json:"moonshotApiKey,omitempty"`
	ZaiAPIKey          string         `json:"zaiApiKey,omitempty"`
	GoogleAPIKey          string         `json:"googleApiKey,omitempty"`
	HeliconeAPIKey        string         `json:"heliconeApiKey,omitempty"`
	DeepInfraAPIKey       string         `json:"deepInfraApiKey,omitempty"`
	NovitaAPIKey          string         `json:"novitaApiKey,omitempty"`
	SiliconFlowAPIKey     string         `json:"siliconFlowApiKey,omitempty"`
	CerebrasAPIKey        string         `json:"cerebrasApiKey,omitempty"`
	SambaNovaAPIKey       string         `json:"sambaNovaApiKey,omitempty"`
	AzureOpenAIAPIKey    string         `json:"azureOpenAIApiKey,omitempty"`
	AzureOpenAIEndpoint  string         `json:"azureOpenAIEndpoint,omitempty"` // e.g. https://YOUR_RESOURCE.openai.azure.com/openai/deployments/YOUR_DEPLOYMENT
	VertexAPIKey         string         `json:"vertexApiKey,omitempty"`
	VertexAPIURL         string         `json:"vertexApiUrl,omitempty"` // e.g. https://REGION-aiplatform.googleapis.com/v1/projects/PROJECT/locations/REGION
	ChatJimmyAPIURL      string         `json:"chatjimmyApiUrl,omitempty"` // optional; default https://chatjimmy.ai (no auth)
	OllamaAPIURL         string         `json:"ollamaApiUrl,omitempty"`   // optional; default http://localhost:11434 for local Ollama
	LocalAIAPIURL        string         `json:"localaiApiUrl,omitempty"`  // optional; default http://localhost:8080 for local LocalAI
	LlamaCppAPIURL       string         `json:"llamacppApiUrl,omitempty"` // optional; default http://localhost:8080 for llama-server
	LMStudioAPIURL       string         `json:"lmstudioApiUrl,omitempty"`  // optional; default http://localhost:1234 for LM Studio
	VLLMAPIURL           string         `json:"vllmApiUrl,omitempty"`     // optional; default http://localhost:8000 for vLLM
	JanAIAPIURL          string         `json:"janaiApiUrl,omitempty"`    // optional; default http://localhost:1337 for Jan AI
	G4fAPIURL            string         `json:"g4fApiUrl,omitempty"`      // optional; default http://localhost:13145 for g4f (GPT4Free)
	InferenceModel        string         `json:"inferenceModel"`
	LowComputeModel       string         `json:"lowComputeModel,omitempty"`       // Optional; used when tier is critical/low_compute
	ResourceConstraintMode string         `json:"resourceConstraintMode,omitempty"` // "auto" | "forced_on" | "forced_off"; default "auto"
	MaxTokensPerTurn   int            `json:"maxTokensPerTurn"`
	HeartbeatConfigPath string        `json:"heartbeatConfigPath"`
	DBPath             string         `json:"dbPath"`
	LogLevel           string         `json:"logLevel"`
	WalletAddress      string            `json:"walletAddress"`
	DefaultChain       string            `json:"defaultChain,omitempty"` // CAIP-2, e.g. "eip155:8453"
	IdentityLabels     map[string]string `json:"identityLabels,omitempty"` // HD index -> friendly name for wallet identities
	ChainProviders     map[string]ChainProviderConfig `json:"chainProviders,omitempty"` // Override RPC+USDC per chain
	SkillsDir          string         `json:"skillsDir"`
	Skills             *SkillsConfig  `json:"skills,omitempty"` // Trusted roots, token budget for prompt injection
	MaxChildren        int            `json:"maxChildren"`
	ParentAddress      string         `json:"parentAddress,omitempty"`
	TreasuryPolicy     *TreasuryPolicy `json:"treasuryPolicy,omitempty"`
	ToolsConfigPath   string         `json:"toolsConfigPath,omitempty"` // Path to YAML/JSON tools config
	Tools             []ConfigToolDef `json:"tools,omitempty"`          // Inline tool definitions
	PluginPaths       []string       `json:"pluginPaths,omitempty"`     // Paths to .so plugin dirs
	Tunnel            *TunnelConfig  `json:"tunnel,omitempty"`           // Tunnel providers for expose_port

	// Model list: add, remove, prioritize LLM providers
	Models []LLMModelEntry `json:"models,omitempty"`

	// Social channels (Conway, Telegram, Discord, etc.)
	SocialChannels    []string `json:"socialChannels,omitempty"`    // e.g. ["conway", "telegram"]
	SocialRelayURL    string   `json:"socialRelayUrl,omitempty"`    // For conway channel
	TelegramBotToken       string                       `json:"telegramBotToken,omitempty"`
	DiscordBotToken        string                       `json:"discordBotToken,omitempty"`
	DiscordGuildID         string                       `json:"discordGuildId,omitempty"`
	SlackBotToken          string                       `json:"slackBotToken,omitempty"`
	TelegramAllowedUsers   []string                     `json:"telegramAllowedUsers,omitempty"`   // DM allowlist; empty=deny all, ["*"]=allow all
	TelegramGroups         []string                     `json:"telegramGroups,omitempty"`        // Group allowlist; ["*"]=all groups; empty=none
	TelegramGroupsConfig   map[string]TelegramGroupCfg  `json:"telegramGroupsConfig,omitempty"` // Per-group: requireMention, etc.
	TelegramRequireMention *bool                        `json:"telegramRequireMention,omitempty"` // In groups, only respond when @mentioned; nil = true (default)
	DiscordAllowedUsers    []string                     `json:"discordAllowedUsers,omitempty"`    // empty=deny all, ["*"]=allow all; else user IDs/usernames
	DiscordAllowedChannels []string                     `json:"discordAllowedChannels,omitempty"` // empty=all channels; else channel IDs to poll
	DiscordMentionOnly     bool                         `json:"discordMentionOnly,omitempty"`
	DiscordListenToBots    bool                         `json:"discordListenToBots,omitempty"`    // default false: ignore bot messages
	DiscordMediaMaxMb      int                          `json:"discordMediaMaxMb,omitempty"`     // max upload size in MB (default 8, OpenClaw-aligned)

	// Soul config: personality, system prompt, tone, behavioral constraints
	Soul *SoulConfig `json:"soul,omitempty"`

	// TestLatencyCooldownSeconds: minimum seconds between test-latency API requests (default 120)
	TestLatencyCooldownSeconds int `json:"testLatencyCooldownSeconds,omitempty"`

	// Token limits for prefill cap safety (Groq ~6k–8k). Defaults: 5500, 12, 4500.
	MaxInputTokens  int `json:"maxInputTokens,omitempty"`  // Safe threshold before truncation
	MaxHistoryTurns int `json:"maxHistoryTurns,omitempty"` // Max history turns when truncating
	WarnAtTokens    int `json:"warnAtTokens,omitempty"`    // Log warning when input exceeds this

	// PromptVersion: "v1" = versioned templates + CoT forcing; empty = legacy ad-hoc prompts.
	PromptVersion string `json:"promptVersion,omitempty"`

	// Memory: auto-ingestion and consolidation (5-tier memory pipeline).
	Memory *MemoryConfig `json:"memory,omitempty"`
}

// MemoryConfig holds automatic memory ingestion settings.
type MemoryConfig struct {
	AutoIngest *MemoryAutoIngestConfig `json:"autoIngest,omitempty"`
}

// MemoryAutoIngestConfig configures the per-turn extraction and background consolidation.
type MemoryAutoIngestConfig struct {
	Enabled                     bool   `json:"enabled"`
	CheapModel                  string `json:"cheapModel,omitempty"`
	ConsolidationIntervalMinutes int    `json:"consolidationIntervalMinutes,omitempty"`
	MaxCandidatesPerBatch       int    `json:"maxCandidatesPerBatch,omitempty"`
}

// TelegramGroupCfg is per-group Telegram config (OpenClaw-style).
type TelegramGroupCfg struct {
	RequireMention bool `json:"requireMention,omitempty"` // Only respond when bot is @mentioned
}

// SoulConfig defines the agent's personality, system prompt, tone, and behavioral constraints.
// Shapes how the agent presents itself and responds.
type SoulConfig struct {
	// SystemPrompt is the base instruction for the agent (used with or instead of genesisPrompt).
	SystemPrompt string `json:"systemPrompt,omitempty"`
	// Personality describes agent traits (e.g. "helpful, analytical, curious").
	Personality string `json:"personality,omitempty"`
	// Tone describes communication style (e.g. "formal", "casual", "professional").
	Tone string `json:"tone,omitempty"`
	// BehavioralConstraints are rules the agent must follow (e.g. "Never disclose private keys").
	BehavioralConstraints []string `json:"behavioralConstraints,omitempty"`
	// SystemPromptVersions keeps the last 30 system prompts (newest first) for history/rollback.
	SystemPromptVersions []string `json:"systemPromptVersions,omitempty"`
}

// ChainProviderConfig configures RPC and USDC contract for a chain (USDC balance check).
type ChainProviderConfig struct {
	RPCURL      string `json:"rpcUrl"`
	USDCAddress string `json:"usdcAddress"`
}

// TunnelConfig configures tunnel providers (expose_port tool).
type TunnelConfig struct {
	DefaultProvider string                    `json:"defaultProvider,omitempty"` // e.g. "bore"
	Providers      map[string]TunnelProviderConfig `json:"providers,omitempty"`
}

// TunnelProviderConfig is per-provider config.
type TunnelProviderConfig struct {
	Enabled      bool   `json:"enabled"`
	StartCommand string `json:"startCommand,omitempty"` // for custom: "bore local {port} --to bore.pub"
	URLPattern   string `json:"urlPattern,omitempty"`
	Token        string `json:"token,omitempty"`     // Cloudflare tunnel token/credential
	AuthToken    string `json:"authToken,omitempty"` // ngrok authtoken (API key for agent)
	AuthKey      string `json:"authKey,omitempty"`   // Tailscale auth key (tskey-auth...)
	Domain       string `json:"domain,omitempty"`    // ngrok custom domain
	Hostname     string `json:"hostname,omitempty"`  // Tailscale hostname
	Funnel       bool   `json:"funnel,omitempty"`    // Tailscale funnel (public HTTPS)
}

// SkillsConfig configures skill loading (trusted roots, token budget).
type SkillsConfig struct {
	TrustedRoots   []string       `json:"trustedRoots,omitempty"`   // Paths under which skill dirs are allowed (~/.automaton/skills, workspace/skills)
	TokenBudgetMax int            `json:"tokenBudgetMax,omitempty"` // Max chars for skills block in prompt (default 2000)
	Registry       *RegistryConfig `json:"registry,omitempty"`      // ClawHub registry for install_skill from registry
}

// RegistryConfig configures the ClawHub skill registry.
type RegistryConfig struct {
	Enabled       bool   `json:"enabled,omitempty"`       // Enable registry installs (default true when registry URL set)
	URL           string `json:"url,omitempty"`           // Registry base URL (default https://clawhub.ai)
	TimeoutSec    int    `json:"timeoutSec,omitempty"`   // HTTP timeout in seconds (default 30)
}

// ConfigToolDef defines a tool loaded from config (extension point).
type ConfigToolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  string `json:"parameters"` // JSON schema string
	Type        string `json:"type"`       // "config", "shell"
	Command     string `json:"command,omitempty"` // Optional shell command template
}

// LLMModelEntry configures a single LLM provider/model for the model list.
// Used for add, remove, prioritize, and per-model settings.
type LLMModelEntry struct {
	ID           string `json:"id"`                     // Unique id; generated on add
	Provider     string `json:"provider"`               // openai, conway, groq, chatjimmy, etc.
	ModelID      string `json:"modelId"`                // API model ID (e.g. gpt-4o, llama-3.3-70b-versatile)
	APIKey       string `json:"apiKey,omitempty"`       // Stored; masked in API responses
	ContextLimit int    `json:"contextLimit,omitempty"` // Max context tokens; 0 = use default
	CostCapCents int    `json:"costCapCents,omitempty"` // Daily cost cap per model; 0 = no cap
	Priority     int    `json:"priority"`              // Lower = higher priority; 0 = first
	Enabled      bool   `json:"enabled"`                // When false, skipped in selection
}

// TreasuryPolicy defines financial limits.
type TreasuryPolicy struct {
	MaxSingleTransferCents   int      `json:"maxSingleTransferCents"`
	MaxHourlyTransferCents   int      `json:"maxHourlyTransferCents"`
	MaxDailyTransferCents    int      `json:"maxDailyTransferCents"`
	MinReserveCents          int      `json:"minReserveCents"`
	X402AllowedDomains       []string `json:"x402AllowedDomains"`
	InferenceDailyBudgetCents int     `json:"inferenceDailyBudgetCents"`
}

// DefaultTreasuryPolicy returns default treasury limits.
func DefaultTreasuryPolicy() TreasuryPolicy {
	return TreasuryPolicy{
		MaxSingleTransferCents:    5000,
		MaxHourlyTransferCents:    10000,
		MaxDailyTransferCents:     50000,
		MinReserveCents:           100,
		X402AllowedDomains:        []string{"api.conway.tech"},
		InferenceDailyBudgetCents:  5000,
	}
}

// ToolCall represents an agent-requested tool invocation.
type ToolCall struct {
	ID   string            `json:"id"`
	Name string            `json:"name"`
	Args map[string]any    `json:"arguments"`
}

// PolicyDecision records allow/deny for audit.
type PolicyDecision struct {
	ID         string   `json:"id"`
	ToolName   string   `json:"toolName"`
	Decision   string   `json:"decision"` // allow, deny, quarantine
	RiskLevel  RiskLevel `json:"riskLevel"`
	Reason     string   `json:"reason"`
}
