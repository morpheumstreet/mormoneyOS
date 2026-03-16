package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/identity"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
	"gopkg.in/yaml.v3"
)

const (
	configFilename = "automaton.json"
)

// GetAutomatonDir returns ~/.automaton or AUTOMATON_DIR.
func GetAutomatonDir() string {
	return identity.GetAutomatonDir()
}

// GetConfigPath returns the full path to automaton.json.
func GetConfigPath() string {
	return filepath.Join(GetAutomatonDir(), configFilename)
}

// ResolvePath expands ~ to home directory.
func ResolvePath(p string) string {
	if len(p) > 0 && p[0] == '~' {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[1:])
	}
	return p
}

// Load reads and merges config with defaults.
func Load() (*types.AutomatonConfig, error) {
	path := GetConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No config yet
		}
		return nil, err
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	cfg := defaultConfig()

	// Apply raw overrides (simplified merge)
	if v, ok := raw["name"].(string); ok {
		cfg.Name = v
	}
	if v, ok := raw["genesisPrompt"].(string); ok {
		cfg.GenesisPrompt = v
	}
	if v, ok := raw["creatorAddress"].(string); ok {
		cfg.CreatorAddress = v
	}
	if v, ok := raw["sandboxId"].(string); ok {
		cfg.SandboxID = v
	}
	if v, ok := raw["conwayApiUrl"].(string); ok && v != "" {
		cfg.ConwayAPIURL = v
	}
	if v, ok := raw["conwayApiKey"].(string); ok {
		cfg.ConwayAPIKey = v
	}
	if v, ok := raw["provider"].(string); ok && v != "" {
		cfg.Provider = v
	}
	if v, ok := raw["openaiApiKey"].(string); ok {
		cfg.OpenAIAPIKey = v
	}
	if v, ok := raw["anthropicApiKey"].(string); ok {
		cfg.AnthropicAPIKey = v
	}
	if v, ok := raw["groqApiKey"].(string); ok {
		cfg.GroqAPIKey = v
	}
	if v, ok := raw["mistralApiKey"].(string); ok {
		cfg.MistralAPIKey = v
	}
	if v, ok := raw["deepSeekApiKey"].(string); ok {
		cfg.DeepSeekAPIKey = v
	}
	if v, ok := raw["openRouterApiKey"].(string); ok {
		cfg.OpenRouterAPIKey = v
	}
	if v, ok := raw["xaiApiKey"].(string); ok {
		cfg.XAIAPIKey = v
	}
	if v, ok := raw["togetherApiKey"].(string); ok {
		cfg.TogetherAPIKey = v
	}
	if v, ok := raw["fireworksApiKey"].(string); ok {
		cfg.FireworksAPIKey = v
	}
	if v, ok := raw["perplexityApiKey"].(string); ok {
		cfg.PerplexityAPIKey = v
	}
	if v, ok := raw["cohereApiKey"].(string); ok {
		cfg.CohereAPIKey = v
	}
	if v, ok := raw["qwenApiKey"].(string); ok {
		cfg.QwenAPIKey = v
	}
	if v, ok := raw["moonshotApiKey"].(string); ok {
		cfg.MoonshotAPIKey = v
	}
	if v, ok := raw["zaiApiKey"].(string); ok {
		cfg.ZaiAPIKey = v
	}
	if v, ok := raw["googleApiKey"].(string); ok {
		cfg.GoogleAPIKey = v
	}
	if v, ok := raw["heliconeApiKey"].(string); ok {
		cfg.HeliconeAPIKey = v
	}
	if v, ok := raw["deepInfraApiKey"].(string); ok {
		cfg.DeepInfraAPIKey = v
	}
	if v, ok := raw["novitaApiKey"].(string); ok {
		cfg.NovitaAPIKey = v
	}
	if v, ok := raw["siliconFlowApiKey"].(string); ok {
		cfg.SiliconFlowAPIKey = v
	}
	if v, ok := raw["cerebrasApiKey"].(string); ok {
		cfg.CerebrasAPIKey = v
	}
	if v, ok := raw["sambaNovaApiKey"].(string); ok {
		cfg.SambaNovaAPIKey = v
	}
	if v, ok := raw["azureOpenAIApiKey"].(string); ok {
		cfg.AzureOpenAIAPIKey = v
	}
	if v, ok := raw["azureOpenAIEndpoint"].(string); ok && v != "" {
		cfg.AzureOpenAIEndpoint = v
	}
	if v, ok := raw["vertexApiKey"].(string); ok {
		cfg.VertexAPIKey = v
	}
	if v, ok := raw["vertexApiUrl"].(string); ok && v != "" {
		cfg.VertexAPIURL = v
	}
	if v, ok := raw["chatjimmyApiUrl"].(string); ok && v != "" {
		cfg.ChatJimmyAPIURL = v
	}
	if v, ok := raw["ollamaApiUrl"].(string); ok && v != "" {
		cfg.OllamaAPIURL = v
	}
	if v, ok := raw["localaiApiUrl"].(string); ok && v != "" {
		cfg.LocalAIAPIURL = v
	}
	if v, ok := raw["llamacppApiUrl"].(string); ok && v != "" {
		cfg.LlamaCppAPIURL = v
	}
	if v, ok := raw["lmstudioApiUrl"].(string); ok && v != "" {
		cfg.LMStudioAPIURL = v
	}
	if v, ok := raw["vllmApiUrl"].(string); ok && v != "" {
		cfg.VLLMAPIURL = v
	}
	if v, ok := raw["janaiApiUrl"].(string); ok && v != "" {
		cfg.JanAIAPIURL = v
	}
	if v, ok := raw["g4fApiUrl"].(string); ok && v != "" {
		cfg.G4fAPIURL = v
	}
	if v, ok := raw["inferenceModel"].(string); ok && v != "" {
		cfg.InferenceModel = v
	}
	if v, ok := raw["lowComputeModel"].(string); ok {
		cfg.LowComputeModel = v
	}
	if v, ok := raw["resourceConstraintMode"].(string); ok && (v == "auto" || v == "forced_on" || v == "forced_off") {
		cfg.ResourceConstraintMode = v
	}
	if v, ok := raw["dbPath"].(string); ok && v != "" {
		cfg.DBPath = ResolvePath(v)
	} else {
		cfg.DBPath = filepath.Join(GetAutomatonDir(), "state.db")
	}
	if v, ok := raw["walletAddress"].(string); ok {
		cfg.WalletAddress = v
	}
	if v, ok := raw["defaultChain"].(string); ok && v != "" {
		cfg.DefaultChain = v
	}
	if m, ok := raw["identityLabels"].(map[string]any); ok {
		cfg.IdentityLabels = make(map[string]string)
		for k, v := range m {
			if s, ok := v.(string); ok {
				cfg.IdentityLabels[k] = s
			}
		}
	}
	if prov, ok := raw["chainProviders"].(map[string]any); ok {
		cfg.ChainProviders = make(map[string]types.ChainProviderConfig)
		for chain, pv := range prov {
			pm, ok := pv.(map[string]any)
			if !ok {
				continue
			}
			pc := types.ChainProviderConfig{}
			if v, ok := pm["rpcUrl"].(string); ok {
				pc.RPCURL = v
			}
			if v, ok := pm["usdcAddress"].(string); ok {
				pc.USDCAddress = v
			}
			if pc.RPCURL != "" && pc.USDCAddress != "" {
				cfg.ChainProviders[chain] = pc
			}
		}
	}
	if v, ok := raw["maxChildren"].(float64); ok && v > 0 {
		cfg.MaxChildren = int(v)
	}
	if v, ok := raw["toolsConfigPath"].(string); ok && v != "" {
		cfg.ToolsConfigPath = ResolvePath(v)
	}
	if v, ok := raw["pluginPaths"].([]any); ok {
		for _, p := range v {
			if s, ok := p.(string); ok && s != "" {
				cfg.PluginPaths = append(cfg.PluginPaths, ResolvePath(s))
			}
		}
	}

	// Inline tools array
	if arr, ok := raw["tools"].([]any); ok {
		for _, a := range arr {
			if m, ok := a.(map[string]any); ok {
				def := types.ConfigToolDef{}
				if v, ok := m["name"].(string); ok {
					def.Name = v
				}
				if v, ok := m["description"].(string); ok {
					def.Description = v
				}
				if v, ok := m["parameters"].(string); ok {
					def.Parameters = v
				} else if v, ok := m["parameters"].(map[string]any); ok {
					b, _ := json.Marshal(v)
					def.Parameters = string(b)
				}
				if v, ok := m["type"].(string); ok {
					def.Type = v
				}
				if v, ok := m["command"].(string); ok {
					def.Command = v
				}
				if def.Name != "" {
					if def.Parameters == "" {
						def.Parameters = `{"type":"object","properties":{},"required":[]}`
					}
					cfg.Tools = append(cfg.Tools, def)
				}
			}
		}
	}

	// Load tools from toolsConfigPath if set (and not already loaded inline)
	if cfg.ToolsConfigPath != "" && len(cfg.Tools) == 0 {
		tools, err := LoadToolsFromFile(cfg.ToolsConfigPath)
		if err == nil {
			cfg.Tools = tools
		}
	}

	// Treasury policy merge
	if tp, ok := raw["treasuryPolicy"].(map[string]any); ok {
		cfg.TreasuryPolicy = mergeTreasuryPolicy(cfg.TreasuryPolicy, tp)
	}

	// Identity fallbacks: wallet address from identity if not in config
	if cfg.WalletAddress == "" && identity.WalletExists() {
		cfg.WalletAddress = identity.GetWalletAddress()
	}
	if cfg.DefaultChain == "" {
		cfg.DefaultChain = identity.DefaultChainBase
	}
	if cfg.ConwayAPIKey == "" {
		cfg.ConwayAPIKey = identity.LoadAPIKeyFromConfig()
	}

	// Tunnel config
	if tc, ok := raw["tunnel"].(map[string]any); ok {
		cfg.Tunnel = mergeTunnelConfig(cfg.Tunnel, tc)
	}

	// Model list
	if arr, ok := raw["models"].([]any); ok {
		for _, a := range arr {
			if m, ok := a.(map[string]any); ok {
				ent := types.LLMModelEntry{Enabled: true}
				if v, ok := m["id"].(string); ok {
					ent.ID = v
				}
				if v, ok := m["provider"].(string); ok {
					ent.Provider = v
				}
				if v, ok := m["modelId"].(string); ok {
					ent.ModelID = v
				}
				if v, ok := m["apiKey"].(string); ok {
					ent.APIKey = v
				}
				if v, ok := m["contextLimit"].(float64); ok && v > 0 {
					ent.ContextLimit = int(v)
				}
				if v, ok := m["costCapCents"].(float64); ok && v >= 0 {
					ent.CostCapCents = int(v)
				}
				if v, ok := m["priority"].(float64); ok {
					ent.Priority = int(v)
				}
				if v, ok := m["enabled"].(bool); ok {
					ent.Enabled = v
				}
				if ent.Provider != "" && ent.ModelID != "" {
					if ent.ID == "" {
						ent.ID = ent.Provider + "_" + ent.ModelID
					}
					cfg.Models = append(cfg.Models, ent)
				}
			}
		}
	}

	// Social channels
	if arr, ok := raw["socialChannels"].([]any); ok {
		for _, a := range arr {
			if s, ok := a.(string); ok && s != "" {
				cfg.SocialChannels = append(cfg.SocialChannels, s)
			}
		}
	}
	if v, ok := raw["socialRelayUrl"].(string); ok && v != "" {
		cfg.SocialRelayURL = v
	}
	if v, ok := raw["telegramBotToken"].(string); ok {
		cfg.TelegramBotToken = v
	}
	if v, ok := raw["discordBotToken"].(string); ok {
		cfg.DiscordBotToken = v
	}
	if v, ok := raw["discordGuildId"].(string); ok {
		cfg.DiscordGuildID = v
	}
	if v, ok := raw["slackBotToken"].(string); ok {
		cfg.SlackBotToken = v
	}
	if arr, ok := raw["telegramAllowedUsers"].([]any); ok {
		for _, a := range arr {
			if s, ok := a.(string); ok {
				cfg.TelegramAllowedUsers = append(cfg.TelegramAllowedUsers, s)
			}
		}
	}
	// OpenClaw alias: allowFrom
	if arr, ok := raw["telegramAllowFrom"].([]any); ok && len(cfg.TelegramAllowedUsers) == 0 {
		for _, a := range arr {
			if s, ok := a.(string); ok {
				cfg.TelegramAllowedUsers = append(cfg.TelegramAllowedUsers, s)
			}
		}
	}
	if arr, ok := raw["telegramGroups"].([]any); ok {
		for _, a := range arr {
			if s, ok := a.(string); ok {
				cfg.TelegramGroups = append(cfg.TelegramGroups, s)
			}
		}
	}
	if m, ok := raw["telegramGroupsConfig"].(map[string]any); ok {
		if cfg.TelegramGroupsConfig == nil {
			cfg.TelegramGroupsConfig = make(map[string]types.TelegramGroupCfg)
		}
		for k, v := range m {
			if sub, ok := v.(map[string]any); ok {
				var gc types.TelegramGroupCfg
				if b, ok := sub["requireMention"].(bool); ok {
					gc.RequireMention = b
				}
				cfg.TelegramGroupsConfig[k] = gc
			}
		}
	}
	if v, ok := raw["telegramRequireMention"].(bool); ok {
		cfg.TelegramRequireMention = &v
	}
	if arr, ok := raw["discordAllowedUsers"].([]any); ok {
		for _, a := range arr {
			if s, ok := a.(string); ok {
				cfg.DiscordAllowedUsers = append(cfg.DiscordAllowedUsers, s)
			}
		}
	}
	// OpenClaw alias: allowFrom
	if arr, ok := raw["discordAllowFrom"].([]any); ok && len(cfg.DiscordAllowedUsers) == 0 {
		for _, a := range arr {
			if s, ok := a.(string); ok {
				cfg.DiscordAllowedUsers = append(cfg.DiscordAllowedUsers, s)
			}
		}
	}
	if arr, ok := raw["discordAllowedChannels"].([]any); ok {
		for _, a := range arr {
			if s, ok := a.(string); ok && s != "" {
				cfg.DiscordAllowedChannels = append(cfg.DiscordAllowedChannels, s)
			}
		}
	}
	if v, ok := raw["discordMentionOnly"].(bool); ok {
		cfg.DiscordMentionOnly = v
	}
	if v, ok := raw["discordListenToBots"].(bool); ok {
		cfg.DiscordListenToBots = v
	}
	if v, ok := raw["discordMediaMaxMb"].(float64); ok && v > 0 {
		cfg.DiscordMediaMaxMb = int(v)
	} else if v, ok := raw["discordMediaMaxMb"].(int); ok && v > 0 {
		cfg.DiscordMediaMaxMb = v
	}

	// Soul config (personality, system prompt, tone, behavioral constraints)
	if sc, ok := raw["soul"].(map[string]any); ok {
		cfg.Soul = mergeSoulConfig(cfg.Soul, sc)
	}

	// Test-latency cooldown (seconds between validate requests; default 120)
	if v, ok := raw["testLatencyCooldownSeconds"].(float64); ok && v > 0 {
		cfg.TestLatencyCooldownSeconds = int(v)
	} else if v, ok := raw["testLatencyCooldownSeconds"].(int); ok && v > 0 {
		cfg.TestLatencyCooldownSeconds = v
	}
	if v, ok := raw["maxInputTokens"].(float64); ok && v > 0 {
		cfg.MaxInputTokens = int(v)
	} else if v, ok := raw["maxInputTokens"].(int); ok && v > 0 {
		cfg.MaxInputTokens = v
	} else if v, ok := raw["max_input_tokens"].(float64); ok && v > 0 {
		cfg.MaxInputTokens = int(v)
	} else if v, ok := raw["max_input_tokens"].(int); ok && v > 0 {
		cfg.MaxInputTokens = v
	}
	if v, ok := raw["maxHistoryTurns"].(float64); ok && v > 0 {
		cfg.MaxHistoryTurns = int(v)
	} else if v, ok := raw["maxHistoryTurns"].(int); ok && v > 0 {
		cfg.MaxHistoryTurns = v
	} else if v, ok := raw["max_history_turns"].(float64); ok && v > 0 {
		cfg.MaxHistoryTurns = int(v)
	} else if v, ok := raw["max_history_turns"].(int); ok && v > 0 {
		cfg.MaxHistoryTurns = v
	}
	if v, ok := raw["warnAtTokens"].(float64); ok && v > 0 {
		cfg.WarnAtTokens = int(v)
	} else if v, ok := raw["warnAtTokens"].(int); ok && v > 0 {
		cfg.WarnAtTokens = v
	} else if v, ok := raw["warn_at_tokens"].(float64); ok && v > 0 {
		cfg.WarnAtTokens = int(v)
	} else if v, ok := raw["warn_at_tokens"].(int); ok && v > 0 {
		cfg.WarnAtTokens = v
	}

	// Skills config (trusted roots for install_skill, token budget for prompt)
	if sc, ok := raw["skills"].(map[string]any); ok {
		cfg.Skills = &types.SkillsConfig{TokenBudgetMax: 2000}
		if arr, ok := sc["trustedRoots"].([]any); ok {
			for _, a := range arr {
				if s, ok := a.(string); ok && s != "" {
					cfg.Skills.TrustedRoots = append(cfg.Skills.TrustedRoots, ResolvePath(s))
				}
			}
		}
		if v, ok := sc["tokenBudgetMax"].(float64); ok && v > 0 {
			cfg.Skills.TokenBudgetMax = int(v)
		}
		if rc, ok := sc["registry"].(map[string]any); ok {
			cfg.Skills.Registry = &types.RegistryConfig{}
			if v, ok := rc["enabled"].(bool); ok {
				cfg.Skills.Registry.Enabled = v
			} else {
				cfg.Skills.Registry.Enabled = true
			}
			if v, ok := rc["url"].(string); ok && v != "" {
				cfg.Skills.Registry.URL = v
			} else {
				cfg.Skills.Registry.URL = "https://clawhub.ai"
			}
			if v, ok := rc["timeoutSec"].(float64); ok && v > 0 {
				cfg.Skills.Registry.TimeoutSec = int(v)
			}
		}
	}
	ensureSkillsConfig(cfg)

	return cfg, nil
}

func ensureSkillsConfig(cfg *types.AutomatonConfig) {
	if cfg != nil && cfg.Skills == nil {
		cfg.Skills = defaultSkillsConfig()
	}
}

func defaultSkillsConfig() *types.SkillsConfig {
	return &types.SkillsConfig{
		TrustedRoots:   []string{ResolvePath("~/.automaton/skills")},
		TokenBudgetMax: 2000,
	}
}

func mergeSoulConfig(base *types.SoulConfig, over map[string]any) *types.SoulConfig {
	out := &types.SoulConfig{}
	if base != nil {
		out.SystemPrompt = base.SystemPrompt
		out.Personality = base.Personality
		out.Tone = base.Tone
		if len(base.BehavioralConstraints) > 0 {
			out.BehavioralConstraints = append([]string{}, base.BehavioralConstraints...)
		}
		if len(base.SystemPromptVersions) > 0 {
			out.SystemPromptVersions = append([]string{}, base.SystemPromptVersions...)
		}
	}
	if v, ok := over["systemPrompt"].(string); ok {
		out.SystemPrompt = v
	}
	if v, ok := over["personality"].(string); ok {
		out.Personality = v
	}
	if v, ok := over["tone"].(string); ok {
		out.Tone = v
	}
	if arr, ok := over["behavioralConstraints"].([]any); ok {
		out.BehavioralConstraints = make([]string, 0, len(arr))
		for _, a := range arr {
			if s, ok := a.(string); ok && s != "" {
				out.BehavioralConstraints = append(out.BehavioralConstraints, s)
			}
		}
	}
	if arr, ok := over["systemPromptVersions"].([]any); ok {
		out.SystemPromptVersions = make([]string, 0, len(arr))
		for _, a := range arr {
			if s, ok := a.(string); ok && s != "" {
				out.SystemPromptVersions = append(out.SystemPromptVersions, s)
			}
		}
	}
	return out
}

func mergeTunnelConfig(base *types.TunnelConfig, over map[string]any) *types.TunnelConfig {
	out := &types.TunnelConfig{DefaultProvider: "bore", Providers: make(map[string]types.TunnelProviderConfig)}
	if base != nil {
		out.DefaultProvider = base.DefaultProvider
		for k, v := range base.Providers {
			out.Providers[k] = v
		}
	}
	if v, ok := over["defaultProvider"].(string); ok && v != "" {
		out.DefaultProvider = v
	}
	if prov, ok := over["providers"].(map[string]any); ok {
		for name, pv := range prov {
			pm, ok := pv.(map[string]any)
			if !ok {
				continue
			}
			pc := types.TunnelProviderConfig{}
			if v, ok := pm["enabled"].(bool); ok {
				pc.Enabled = v
			} else {
				pc.Enabled = true // default enabled when present
			}
			if v, ok := pm["startCommand"].(string); ok {
				pc.StartCommand = v
			}
			if v, ok := pm["urlPattern"].(string); ok {
				pc.URLPattern = v
			}
			if v, ok := pm["token"].(string); ok {
				pc.Token = os.ExpandEnv(v)
			}
			if v, ok := pm["authToken"].(string); ok {
				pc.AuthToken = os.ExpandEnv(v)
			}
			if v, ok := pm["authKey"].(string); ok {
				pc.AuthKey = os.ExpandEnv(v)
			}
			if v, ok := pm["domain"].(string); ok {
				pc.Domain = v
			}
			if v, ok := pm["hostname"].(string); ok {
				pc.Hostname = v
			}
			if v, ok := pm["funnel"].(bool); ok {
				pc.Funnel = v
			}
			out.Providers[name] = pc
		}
	}
	return out
}

// LoadToolsFromFile loads tool definitions from a JSON or YAML file.
func LoadToolsFromFile(path string) ([]types.ConfigToolDef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw struct {
		Tools []map[string]any `json:"tools" yaml:"tools"`
	}
	path = strings.ToLower(path)
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
	} else {
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
	}
	var out []types.ConfigToolDef
	for _, m := range raw.Tools {
		def := types.ConfigToolDef{}
		if v, ok := m["name"].(string); ok {
			def.Name = v
		}
		if v, ok := m["description"].(string); ok {
			def.Description = v
		}
		if v, ok := m["parameters"].(string); ok {
			def.Parameters = v
		} else if v, ok := m["parameters"].(map[string]any); ok {
			b, _ := json.Marshal(v)
			def.Parameters = string(b)
		}
		if v, ok := m["type"].(string); ok {
			def.Type = v
		}
		if v, ok := m["command"].(string); ok {
			def.Command = v
		}
		if def.Name != "" {
			if def.Parameters == "" {
				def.Parameters = `{"type":"object","properties":{},"required":[]}`
			}
			out = append(out, def)
		}
	}
	return out, nil
}

// DefaultConfig returns a new config with default values. Used when no config file exists yet.
func DefaultConfig() *types.AutomatonConfig {
	return defaultConfig()
}

func defaultConfig() *types.AutomatonConfig {
	tp := types.DefaultTreasuryPolicy()
	return &types.AutomatonConfig{
		ConwayAPIURL:               "https://api.conway.tech",
		Provider:                   "chatjimmy",
		InferenceModel:             "llama3.1-8B",
		MaxTokensPerTurn:           4096,
		HeartbeatConfigPath:        ResolvePath("~/.automaton/heartbeat.yml"),
		DBPath:                     filepath.Join(GetAutomatonDir(), "state.db"),
		LogLevel:                   "info",
		MaxChildren:                3,
		TreasuryPolicy:             &tp,
		TestLatencyCooldownSeconds: 120,
		MaxInputTokens:             5500,
		MaxHistoryTurns:            12,
		WarnAtTokens:               4500,
	}
}

func mergeTreasuryPolicy(base *types.TreasuryPolicy, over map[string]any) *types.TreasuryPolicy {
	if base == nil {
		b := types.DefaultTreasuryPolicy()
		base = &b
	}
	out := *base
	if v, ok := over["maxSingleTransferCents"].(float64); ok && v >= 0 {
		out.MaxSingleTransferCents = int(v)
	}
	if v, ok := over["maxHourlyTransferCents"].(float64); ok && v >= 0 {
		out.MaxHourlyTransferCents = int(v)
	}
	if v, ok := over["maxDailyTransferCents"].(float64); ok && v >= 0 {
		out.MaxDailyTransferCents = int(v)
	}
	if v, ok := over["minReserveCents"].(float64); ok && v >= 0 {
		out.MinReserveCents = int(v)
	}
	return &out
}

// Save writes config to disk.
func Save(cfg *types.AutomatonConfig) error {
	dir := GetAutomatonDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(GetConfigPath(), data, 0600)
}
