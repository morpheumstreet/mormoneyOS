package heartbeat

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/morpheumlabs/mormoneyos-go/internal/social"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

// InboxResult is the outcome of processing one social message.
type InboxResult struct {
	Record     map[string]any // for last_social_check payload
	SendErr    error          // non-nil if Send failed (command reply or ack)
	ShouldWake bool
	WakeMsg    string
}

// ProcessInboxMessage handles one message: Type 2 (programmatic) or Type 1 (LLM).
// Returns the record for status, any send error, and wake signal.
func ProcessInboxMessage(ctx context.Context, tc *TaskContext, m social.InboxMessage, ch social.SocialChannel, agentSleeping bool) InboxResult {
	// Type 2 (programmatic): Fast-reply — immediate reply, no agent, no LLM.
	if fast, norm := social.ClassifyFastReply(m.Content); fast && norm != "" {
		mNorm := m
		mNorm.Content = norm
		resp, parseMode, handled := HandleSocialCommand(ctx, tc, mNorm)
		if handled {
			_ = tc.DB.SetKV("inbox_seen_"+m.ID, "1")
			outMsg := &social.OutboundMessage{
				Content:   resp,
				Recipient: m.ReplyTarget,
				ReplyTo:   m.ID,
				ParseMode: parseMode,
			}
			if _, err := ch.Send(ctx, outMsg); err != nil {
				slog.Warn("social command send failed", "cmd", norm, "channel", m.Channel, "err", err)
				return InboxResult{
					Record:  map[string]any{"id": m.ID, "from": m.Sender, "content": m.Content, "channel": m.Channel, "command": true},
					SendErr: err,
				}
			}
			slog.Info("social command replied", "cmd", norm, "channel", m.Channel)
			return InboxResult{
				Record: map[string]any{"id": m.ID, "from": m.Sender, "content": m.Content, "channel": m.Channel, "command": true},
			}
		}
	}

	// Type 1 (LLM): Non-commands — queue for agent, wake, optional ack.
	record := map[string]any{"id": m.ID, "from": m.Sender, "content": m.Content, "channel": m.Channel}
	if db, ok := tc.DB.(*state.Database); ok {
		seen, _, _ := tc.DB.GetKV("inbox_seen_" + m.ID)
		if seen == "" {
			_ = db.InsertInboxMessage(m.ID, m.Sender, m.Content, "")
			// Store route for fallback reply when LLM fails (channel|recipient)
			_ = tc.DB.SetKV("inbox_route:"+m.ID, m.Channel+"|"+m.ReplyTarget)
			_ = tc.DB.SetKV("inbox_seen_"+m.ID, "1")
			if agentSleeping {
				ack := &social.OutboundMessage{
					Content:   "Message received. Agent will respond when it wakes.",
					Recipient: m.ReplyTarget,
					ReplyTo:   m.ID,
				}
				if _, err := ch.Send(ctx, ack); err != nil {
					return InboxResult{Record: record, SendErr: err, ShouldWake: true, WakeMsg: fmt.Sprintf("New message from %s on %s", m.Sender, m.Channel)}
				}
			}
		}
	}
	return InboxResult{
		Record:     record,
		ShouldWake: true,
		WakeMsg:    fmt.Sprintf("New message from %s on %s", m.Sender, m.Channel),
	}
}
