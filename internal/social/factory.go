package social

import (
	"context"
	"strings"
	"time"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// ChannelStatus describes a social channel for the config UI.
type ChannelStatus struct {
	Key         string
	DisplayName string
	Enabled     bool
	Ready       bool
}

// ConfigField describes a config field for a social channel (API key, URL, etc.).
type ConfigField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Type        string `json:"type"` // "password", "text", "array", "boolean"
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

// ChannelConfigSchema defines config fields per channel.
var channelConfigSchema = map[string][]ConfigField{
	"conway": {
		{Key: "socialRelayUrl", Label: "Relay URL", Type: "text", Required: true, Description: "Conway social relay API URL (HTTPS)"},
	},
	"telegram": {
		{Key: "telegramBotToken", Label: "Bot Token", Type: "password", Required: true, Description: "Telegram Bot API token from @BotFather"},
		{Key: "telegramAllowedUsers", Label: "Allowed Users", Type: "array", Required: false, Description: "Comma-separated usernames or user IDs; empty = allow all"},
	},
	"discord": {
		{Key: "discordBotToken", Label: "Bot Token", Type: "password", Required: true, Description: "Discord bot token from Developer Portal"},
		{Key: "discordGuildId", Label: "Guild ID", Type: "text", Required: false, Description: "Server ID for inbox polling"},
		{Key: "discordAllowedUsers", Label: "Allowed Users", Type: "array", Required: false, Description: "Comma-separated user IDs or usernames"},
		{Key: "discordMentionOnly", Label: "Mention Only", Type: "boolean", Required: false, Description: "Only process messages that @mention the bot"},
	},
}

// GetChannelConfigSchema returns config fields for a channel.
func GetChannelConfigSchema(key string) []ConfigField {
	return channelConfigSchema[key]
}

// GetChannelConfigValues returns current config values for a channel (secrets masked).
func GetChannelConfigValues(key string, cfg *types.AutomatonConfig) map[string]any {
	if cfg == nil {
		return nil
	}
	out := make(map[string]any)
	switch key {
	case "conway":
		if cfg.SocialRelayURL != "" {
			out["socialRelayUrl"] = cfg.SocialRelayURL
		}
	case "telegram":
		if cfg.TelegramBotToken != "" {
			out["telegramBotToken"] = "••••••••"
		}
		if len(cfg.TelegramAllowedUsers) > 0 {
			out["telegramAllowedUsers"] = cfg.TelegramAllowedUsers
		}
	case "discord":
		if cfg.DiscordBotToken != "" {
			out["discordBotToken"] = "••••••••"
		}
		if cfg.DiscordGuildID != "" {
			out["discordGuildId"] = cfg.DiscordGuildID
		}
		if len(cfg.DiscordAllowedUsers) > 0 {
			out["discordAllowedUsers"] = cfg.DiscordAllowedUsers
		}
		out["discordMentionOnly"] = cfg.DiscordMentionOnly
	}
	return out
}

// ApplyChannelConfig updates config with values from the given map for the channel.
// Only updates fields that are present in updates; empty string for password means "keep existing".
func ApplyChannelConfig(cfg *types.AutomatonConfig, key string, updates map[string]any) {
	if cfg == nil || updates == nil {
		return
	}
	switch key {
	case "conway":
		if v, ok := updates["socialRelayUrl"].(string); ok {
			cfg.SocialRelayURL = v
		}
	case "telegram":
		if v, ok := updates["telegramBotToken"].(string); ok && v != "" {
			cfg.TelegramBotToken = v
		}
		if v, ok := updates["telegramAllowedUsers"]; ok {
			switch val := v.(type) {
			case []string:
				cfg.TelegramAllowedUsers = val
			case []any:
				var arr []string
				for _, a := range val {
					if s, ok := a.(string); ok && s != "" {
						arr = append(arr, s)
					}
				}
				cfg.TelegramAllowedUsers = arr
			case string:
				if val != "" {
					cfg.TelegramAllowedUsers = splitCommaTrim(val)
				}
			}
		}
	case "discord":
		if v, ok := updates["discordBotToken"].(string); ok && v != "" {
			cfg.DiscordBotToken = v
		}
		if v, ok := updates["discordGuildId"].(string); ok {
			cfg.DiscordGuildID = v
		}
		if v, ok := updates["discordAllowedUsers"]; ok {
			switch val := v.(type) {
			case []string:
				cfg.DiscordAllowedUsers = val
			case []any:
				var arr []string
				for _, a := range val {
					if s, ok := a.(string); ok && s != "" {
						arr = append(arr, s)
					}
				}
				cfg.DiscordAllowedUsers = arr
			case string:
				if val != "" {
					cfg.DiscordAllowedUsers = splitCommaTrim(val)
				}
			}
		}
		if v, ok := updates["discordMentionOnly"].(bool); ok {
			cfg.DiscordMentionOnly = v
		}
	}
}

// ValidateChannel creates the channel from config and runs HealthCheck.
// Returns nil if validation succeeds.
func ValidateChannel(ctx context.Context, cfg *types.AutomatonConfig, key string) error {
	spec := LookupChannel(key)
	if spec == nil {
		return nil
	}
	ch, err := spec.Constructor(cfg)
	if err != nil {
		return err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return ch.HealthCheck(ctx)
}

func splitCommaTrim(s string) []string {
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ListChannelsWithStatus returns all known channels with enabled/ready status.
func ListChannelsWithStatus(cfg *types.AutomatonConfig) []ChannelStatus {
	enabledSet := make(map[string]bool)
	if cfg != nil {
		for _, k := range cfg.SocialChannels {
			enabledSet[k] = true
		}
	}
	out := make([]ChannelStatus, 0, len(registry))
	for i := range registry {
		spec := &registry[i]
		ready := isChannelReady(spec, cfg)
		out = append(out, ChannelStatus{
			Key:         spec.Key,
			DisplayName: spec.DisplayName,
			Enabled:     enabledSet[spec.Key],
			Ready:       ready,
		})
	}
	return out
}

// NewChannelsFromConfig builds all enabled channels from config.
// Returns map keyed by channel name for tool/heartbeat use.
func NewChannelsFromConfig(cfg *types.AutomatonConfig) map[string]SocialChannel {
	if cfg == nil {
		return nil
	}
	channels := make(map[string]SocialChannel)
	for _, key := range cfg.SocialChannels {
		spec := LookupChannel(key)
		if spec == nil {
			continue
		}
		if !isChannelReady(spec, cfg) {
			continue
		}
		ch, err := spec.Constructor(cfg)
		if err != nil {
			continue
		}
		channels[key] = ch
	}
	return channels
}

func isChannelReady(spec *ChannelSpec, cfg *types.AutomatonConfig) bool {
	if spec.URIConfigKey != "" {
		url := getSocialRelayURL(cfg)
		if url == "" {
			return false
		}
	}
	if spec.TokenConfigKey != "" {
		token := getTokenFromConfig(spec.TokenConfigKey, cfg)
		if token == "" {
			return false
		}
	}
	if spec.Key == "conway" {
		// Conway needs relay URL + wallet address
		if getSocialRelayURL(cfg) == "" || cfg.WalletAddress == "" {
			return false
		}
	}
	return true
}

func getSocialRelayURL(cfg *types.AutomatonConfig) string {
	if cfg == nil {
		return ""
	}
	return cfg.SocialRelayURL
}

func getTokenFromConfig(key string, cfg *types.AutomatonConfig) string {
	if cfg == nil {
		return ""
	}
	switch key {
	case "TelegramBotToken":
		return cfg.TelegramBotToken
	case "DiscordBotToken":
		return cfg.DiscordBotToken
	case "SlackBotToken":
		return cfg.SlackBotToken
	default:
		return ""
	}
}
