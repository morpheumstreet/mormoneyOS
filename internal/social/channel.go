package social

import "context"

// OutboundMessage is the normalized outbound payload.
type OutboundMessage struct {
	Content   string
	Recipient string // Conway: 0x... wallet; Telegram: chat_id; Discord: channel_id
	ThreadID  string // Optional; for threaded replies (Slack thread_ts, Discord thread)
	ReplyTo   string // Optional message ID to reply to
}

// InboxMessage is a normalized inbound message.
type InboxMessage struct {
	ID          string
	Sender      string
	ReplyTarget string // Where to send reply (chat_id, channel_id, 0x...); session routing
	Content     string
	Channel     string // "conway", "telegram", "discord", etc.
	Timestamp   int64
	ThreadID    string
}

// SocialChannel is the minimal interface for all social platforms (Conway, Telegram, Discord, etc.).
type SocialChannel interface {
	Name() string
	Send(ctx context.Context, msg *OutboundMessage) (messageID string, err error)
	Poll(ctx context.Context, cursor string, limit int) ([]InboxMessage, string, error)
	HealthCheck(ctx context.Context) error
}

// Refreshable allows a channel to refresh its auth token and invalidate on 401/403.
// Conway uses wallet signing (no token); Discord/Telegram use TokenManager.
type Refreshable interface {
	GetAuthToken(ctx context.Context) (string, error)
	Invalidate() // Force refresh on next GetAuthToken (call on 401/403)
}

// ManagedChannel extends SocialChannel with token lifecycle support.
type ManagedChannel interface {
	SocialChannel
	Refreshable
}
