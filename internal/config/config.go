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
	if v, ok := raw["provider"].(string); ok {
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
	if v, ok := raw["chatjimmyApiUrl"].(string); ok && v != "" {
		cfg.ChatJimmyAPIURL = v
	}
	if v, ok := raw["inferenceModel"].(string); ok && v != "" {
		cfg.InferenceModel = v
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

	return cfg, nil
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

func defaultConfig() *types.AutomatonConfig {
	tp := types.DefaultTreasuryPolicy()
	return &types.AutomatonConfig{
		ConwayAPIURL:        "https://api.conway.tech",
		InferenceModel:      "gpt-5.2",
		MaxTokensPerTurn:    4096,
		HeartbeatConfigPath: ResolvePath("~/.automaton/heartbeat.yml"),
		DBPath:              filepath.Join(GetAutomatonDir(), "state.db"),
		LogLevel:            "info",
		MaxChildren:         3,
		TreasuryPolicy:      &tp,
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
