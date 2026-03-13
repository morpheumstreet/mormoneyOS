package tools

import (
	"context"

	"github.com/morpheumlabs/mormoneyos-go/internal/social"
)

// SocialChannelAdapter adapts social.SocialChannel to tools.SocialClient for message_child.
// Used when the conway channel is available.
type SocialChannelAdapter struct {
	Channel social.SocialChannel
}

// Send implements SocialClient.
func (a *SocialChannelAdapter) Send(toAddress string, payload string) (string, error) {
	return a.Channel.Send(context.Background(), &social.OutboundMessage{
		Content:   payload,
		Recipient: toAddress,
	})
}
