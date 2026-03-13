package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/social"
)

// SendMessageTool sends a message via a social channel (Conway, Telegram, Discord, etc.).
type SendMessageTool struct {
	Channels map[string]social.SocialChannel
}

func (SendMessageTool) Name() string        { return "send_message" }
func (SendMessageTool) Description() string { return "Send a message via a social channel (Conway, Telegram, Discord). Use channel and recipient; omit channel to use default." }
func (SendMessageTool) Parameters() string {
	return `{"type":"object","properties":{"channel":{"type":"string","description":"Channel: conway, telegram, discord. Omit for default."},"recipient":{"type":"string","description":"Recipient: 0x... for Conway; chat/channel ID for Telegram/Discord"},"to_address":{"type":"string","description":"Alias for recipient (Conway wallet)"},"content":{"type":"string","description":"Message content"},"reply_to":{"type":"string","description":"Optional message ID to reply to"}},"required":["content"]}`
}

func (t *SendMessageTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Channels == nil || len(t.Channels) == 0 {
		return "No social channels configured. Add socialChannels and credentials to config.", nil
	}
	recipient, _ := args["recipient"].(string)
	if recipient == "" {
		recipient, _ = args["to_address"].(string)
	}
	if recipient == "" {
		recipient, _ = args["to"].(string)
	}
	recipient = strings.TrimSpace(recipient)
	if recipient == "" {
		return "", ErrInvalidArgs{Msg: "recipient or to_address required"}
	}
	content, _ := args["content"].(string)
	if content == "" {
		return "", ErrInvalidArgs{Msg: "content required"}
	}
	channelKey, _ := args["channel"].(string)
	channelKey = strings.TrimSpace(channelKey)
	replyTo, _ := args["reply_to"].(string)
	replyTo = strings.TrimSpace(replyTo)

	var ch social.SocialChannel
	if channelKey != "" {
		ch = t.Channels[channelKey]
		if ch == nil {
			return fmt.Sprintf("Channel %q not enabled or not configured.", channelKey), nil
		}
	} else {
		// Use first enabled channel as default
		for _, c := range t.Channels {
			ch = c
			break
		}
	}
	if ch == nil {
		return "No social channel available.", nil
	}

	msg := &social.OutboundMessage{
		Content:   content,
		Recipient: recipient,
		ReplyTo:   replyTo,
	}
	id, err := ch.Send(ctx, msg)
	if err != nil {
		return "", fmt.Errorf("send_message: %w", err)
	}
	return fmt.Sprintf("Message sent (id: %s)", id), nil
}
