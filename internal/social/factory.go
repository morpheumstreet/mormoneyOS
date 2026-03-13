package social

import (
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

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
