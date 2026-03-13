package social

import (
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// ChannelSpec describes a social channel for registry lookup.
type ChannelSpec struct {
	Key            string
	DisplayName    string
	TokenConfigKey string // e.g. "TelegramBotToken"; Conway uses wallet, not token
	URIConfigKey   string // e.g. "SocialRelayURL" for Conway
	Constructor    func(cfg *types.AutomatonConfig) (SocialChannel, error)
}

// registry holds all known channel specs.
var registry = []ChannelSpec{
	{Key: "conway", DisplayName: "Conway", TokenConfigKey: "", URIConfigKey: "SocialRelayURL", Constructor: NewConwayChannel},
	{Key: "telegram", DisplayName: "Telegram", TokenConfigKey: "TelegramBotToken", URIConfigKey: "", Constructor: NewTelegramChannel},
	{Key: "discord", DisplayName: "Discord", TokenConfigKey: "DiscordBotToken", URIConfigKey: "", Constructor: NewDiscordChannel},
}

// LookupChannel returns the spec for a channel key, or nil if not found.
func LookupChannel(key string) *ChannelSpec {
	for i := range registry {
		if registry[i].Key == key {
			return &registry[i]
		}
	}
	return nil
}

// ListChannels returns all registered channel keys.
func ListChannels() []string {
	keys := make([]string, len(registry))
	for i := range registry {
		keys[i] = registry[i].Key
	}
	return keys
}
