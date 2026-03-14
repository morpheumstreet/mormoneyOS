package social

import (
	"context"
	"fmt"
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
		{Key: "telegramAllowedUsers", Label: "Allowed Users (DMs)", Type: "array", Required: false, Description: "empty=deny all; [\"*\"]=allow all; else list of @usernames or user IDs"},
		{Key: "telegramGroups", Label: "Allowed Groups", Type: "array", Required: false, Description: "[\"*\"]=all groups; else list of group chat IDs; empty=no groups"},
		{Key: "telegramGroupsConfig", Label: "Per-Group Config", Type: "object", Required: false, Description: "Map groupId -> { requireMention: bool }"},
		{Key: "telegramRequireMention", Label: "Require @mention in Groups", Type: "boolean", Required: false, Description: "Only respond when bot is @mentioned (default: true)"},
	},
	"discord": {
		{Key: "discordBotToken", Label: "Bot Token", Type: "password", Required: true, Description: "Discord bot token from Developer Portal"},
		{Key: "discordGuildId", Label: "Guild ID", Type: "text", Required: false, Description: "Server ID for inbox polling"},
		{Key: "discordAllowedUsers", Label: "Allowed Users", Type: "array", Required: false, Description: "empty=deny all; [\"*\"]=allow all; else user IDs or usernames (OpenClaw-aligned)"},
		{Key: "discordAllowedChannels", Label: "Allowed Channels", Type: "array", Required: false, Description: "Channel IDs to poll; empty=all channels in guild"},
		{Key: "discordMentionOnly", Label: "Mention Only", Type: "boolean", Required: false, Description: "Only process messages that @mention the bot"},
		{Key: "discordListenToBots", Label: "Listen to Bots", Type: "boolean", Required: false, Description: "Process messages from other bots (default: false)"},
		{Key: "discordMediaMaxMb", Label: "Media Max MB", Type: "text", Required: false, Description: "Max upload size in MB (default 8, for future file attachments)"},
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
		if len(cfg.TelegramGroups) > 0 {
			out["telegramGroups"] = cfg.TelegramGroups
		}
		if len(cfg.TelegramGroupsConfig) > 0 {
			out["telegramGroupsConfig"] = cfg.TelegramGroupsConfig
		}
		if cfg.TelegramRequireMention != nil {
			out["telegramRequireMention"] = *cfg.TelegramRequireMention
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
		if len(cfg.DiscordAllowedChannels) > 0 {
			out["discordAllowedChannels"] = cfg.DiscordAllowedChannels
		}
		out["discordMentionOnly"] = cfg.DiscordMentionOnly
		out["discordListenToBots"] = cfg.DiscordListenToBots
		if cfg.DiscordMediaMaxMb > 0 {
			out["discordMediaMaxMb"] = cfg.DiscordMediaMaxMb
		}
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
		if v, ok := updates["telegramGroups"]; ok {
			switch val := v.(type) {
			case []string:
				cfg.TelegramGroups = val
			case []any:
				var arr []string
				for _, a := range val {
					if s, ok := a.(string); ok && s != "" {
						arr = append(arr, s)
					}
				}
				cfg.TelegramGroups = arr
			case string:
				if val != "" {
					cfg.TelegramGroups = splitCommaTrim(val)
				}
			}
		}
		if v, ok := updates["telegramGroupsConfig"]; ok {
			if m, ok := v.(map[string]any); ok {
				if cfg.TelegramGroupsConfig == nil {
					cfg.TelegramGroupsConfig = make(map[string]types.TelegramGroupCfg)
				}
				for k, sub := range m {
					if sm, ok := sub.(map[string]any); ok {
						var gc types.TelegramGroupCfg
						if b, ok := sm["requireMention"].(bool); ok {
							gc.RequireMention = b
						}
						cfg.TelegramGroupsConfig[k] = gc
					}
				}
			}
		}
		if v, ok := updates["telegramRequireMention"].(bool); ok {
			cfg.TelegramRequireMention = &v
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
					if s, ok := a.(string); ok {
						arr = append(arr, s)
					}
				}
				cfg.DiscordAllowedUsers = arr
			case string:
				cfg.DiscordAllowedUsers = splitCommaTrim(val)
			}
		}
		if v, ok := updates["discordAllowedChannels"]; ok {
			switch val := v.(type) {
			case []string:
				cfg.DiscordAllowedChannels = val
			case []any:
				var arr []string
				for _, a := range val {
					if s, ok := a.(string); ok && s != "" {
						arr = append(arr, s)
					}
				}
				cfg.DiscordAllowedChannels = arr
			case string:
				if val != "" {
					cfg.DiscordAllowedChannels = splitCommaTrim(val)
				}
			}
		}
		if v, ok := updates["discordMentionOnly"].(bool); ok {
			cfg.DiscordMentionOnly = v
		}
		if v, ok := updates["discordListenToBots"].(bool); ok {
			cfg.DiscordListenToBots = v
		}
		if v, ok := updates["discordMediaMaxMb"].(float64); ok && v > 0 {
			cfg.DiscordMediaMaxMb = int(v)
		} else if v, ok := updates["discordMediaMaxMb"].(int); ok && v > 0 {
			cfg.DiscordMediaMaxMb = v
		} else if v, ok := updates["discordMediaMaxMb"].(string); ok && v != "" {
			var n int
			if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
				cfg.DiscordMediaMaxMb = n
			}
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
