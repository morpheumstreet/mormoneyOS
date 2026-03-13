package social

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

const (
	discordAPIBase   = "https://discord.com/api/v10"
	discordMaxContent = 2000 // Discord message content limit
)

// DiscordChannel implements SocialChannel for Discord Bot API (REST).
type DiscordChannel struct {
	token        string
	guildID      string
	botID        string // For mention_only check
	allowedUsers map[string]struct{}
	mentionOnly  bool
}

// NewDiscordChannel creates a Discord channel. Requires DiscordBotToken.
// DiscordGuildID is required for Poll; without it, Poll returns empty.
func NewDiscordChannel(cfg *types.AutomatonConfig) (SocialChannel, error) {
	if cfg == nil || cfg.DiscordBotToken == "" {
		return nil, fmt.Errorf("discord: bot token required")
	}
	ch := &DiscordChannel{
		token:        cfg.DiscordBotToken,
		guildID:      strings.TrimSpace(cfg.DiscordGuildID),
		allowedUsers: make(map[string]struct{}),
		mentionOnly:  cfg.DiscordMentionOnly,
	}
	for _, u := range cfg.DiscordAllowedUsers {
		u = strings.TrimSpace(strings.ToLower(u))
		if u != "" {
			ch.allowedUsers[u] = struct{}{}
		}
	}
	// Resolve bot ID for mention_only; ignore error, we'll skip mention check if unknown
	if id, err := ch.resolveBotID(context.Background()); err == nil {
		ch.botID = id
	}
	return ch, nil
}

func (c *DiscordChannel) Name() string {
	return "discord"
}

func (c *DiscordChannel) Send(ctx context.Context, msg *OutboundMessage) (string, error) {
	if err := ValidateOutbound(msg.Content); err != nil {
		return "", err
	}
	channelID := strings.TrimSpace(msg.Recipient)
	if channelID == "" {
		return "", fmt.Errorf("discord: recipient (channel_id) required")
	}

	content := msg.Content
	if len(content) > discordMaxContent {
		content = content[:discordMaxContent]
	}

	payload := map[string]any{"content": content}
	if msg.ReplyTo != "" {
		payload["message_reference"] = map[string]any{
			"message_id": msg.ReplyTo,
		}
	}
	if msg.ThreadID != "" {
		// For threads, Recipient can be the thread ID; message_reference can include channel_id
		if ref, ok := payload["message_reference"].(map[string]any); ok {
			ref["channel_id"] = channelID
		}
	}

	body, _ := json.Marshal(payload)
	url := discordAPIBase + "/channels/" + channelID + "/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bot "+c.token)

	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("discord send: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(respBody, &errResp)
		if errResp.Message != "" {
			return "", fmt.Errorf("discord send %d: %s", resp.StatusCode, errResp.Message)
		}
		return "", fmt.Errorf("discord send %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("discord: parse response: %w", err)
	}
	return result.ID, nil
}

func (c *DiscordChannel) Poll(ctx context.Context, cursor string, limit int) ([]InboxMessage, string, error) {
	if c.guildID == "" {
		return nil, "", nil
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	channels, err := c.listGuildChannels(ctx)
	if err != nil {
		return nil, "", err
	}

	cursorMap := parseDiscordCursor(cursor)
	allMsgs := make([]discordPollMsg, 0)

	for _, ch := range channels {
		lastID := cursorMap[ch.ID]
		msgs, err := c.getChannelMessages(ctx, ch.ID, lastID, 100)
		if err != nil {
			continue
		}
		for _, m := range msgs {
			allMsgs = append(allMsgs, discordPollMsg{channelID: ch.ID, msg: m})
		}
	}

	// Sort by timestamp ascending (oldest first)
	sort.Slice(allMsgs, func(i, j int) bool {
		return allMsgs[i].msg.Timestamp < allMsgs[j].msg.Timestamp
	})

	// Build next cursor: last message ID per channel (we iterate oldest-first, so last = newest)
	nextCursor := make(map[string]string)
	for _, p := range allMsgs {
		nextCursor[p.channelID] = p.msg.ID
	}

	out := make([]InboxMessage, 0, limit)
	for _, p := range allMsgs {
		if len(out) >= limit {
			break
		}
		m := p.msg
		if m.Content == "" && len(m.Embeds) == 0 {
			continue
		}
		content := m.Content
		for _, e := range m.Embeds {
			if e.Description != "" {
				content += "\n" + e.Description
			}
		}
		content = strings.TrimSpace(content)
		if content == "" {
			continue
		}

		authorID := ""
		if m.Author != nil {
			authorID = m.Author.ID
		}
		senderKey := strings.ToLower(authorID)
		if m.Author != nil && m.Author.Username != "" {
			senderKey = strings.ToLower(m.Author.Username)
		}

		if len(c.allowedUsers) > 0 {
			if _, ok := c.allowedUsers[authorID]; !ok {
				if _, ok := c.allowedUsers[senderKey]; !ok {
					continue
				}
			}
		}

		if c.mentionOnly && c.botID != "" {
			mentioned := false
			for _, u := range m.Mentions {
				if u.ID == c.botID {
					mentioned = true
					break
				}
			}
			if !mentioned {
				continue
			}
		}

		threadID := ""
		if m.Thread != nil {
			threadID = m.Thread.ID
		} else if m.ChannelID != "" {
			threadID = m.ChannelID
		}

		out = append(out, InboxMessage{
			ID:          m.ID,
			Sender:      authorID,
			ReplyTarget: p.channelID,
			Content:     content,
			Channel:     "discord",
			Timestamp:   parseDiscordTimestamp(m.Timestamp),
			ThreadID:    threadID,
		})
	}

	return out, encodeDiscordCursor(nextCursor), nil
}

func (c *DiscordChannel) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discordAPIBase+"/users/@me", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bot "+c.token)
	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("discord health: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord health %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (c *DiscordChannel) resolveBotID(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discordAPIBase+"/users/@me", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bot "+c.token)
	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	respBody, _ := io.ReadAll(resp.Body)
	var u struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &u); err != nil {
		return "", err
	}
	return u.ID, nil
}

func (c *DiscordChannel) listGuildChannels(ctx context.Context) ([]discordChannel, error) {
	url := discordAPIBase + "/guilds/" + c.guildID + "/channels"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bot "+c.token)
	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("discord list channels: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord list channels %d", resp.StatusCode)
	}
	respBody, _ := io.ReadAll(resp.Body)
	var channels []discordChannel
	if err := json.Unmarshal(respBody, &channels); err != nil {
		return nil, err
	}
	// Filter to text-like channels: GUILD_TEXT (0), GUILD_ANNOUNCEMENT (5), threads (10, 11, 12)
	var out []discordChannel
	for _, ch := range channels {
		switch ch.Type {
		case 0, 5, 10, 11, 12:
			out = append(out, ch)
		}
	}
	return out, nil
}

func (c *DiscordChannel) getChannelMessages(ctx context.Context, channelID, after string, limit int) ([]discordMessage, error) {
	url := discordAPIBase + "/channels/" + channelID + "/messages?limit=" + fmt.Sprintf("%d", limit)
	if after != "" {
		url += "&after=" + after
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bot "+c.token)
	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	respBody, _ := io.ReadAll(resp.Body)
	var msgs []discordMessage
	if err := json.Unmarshal(respBody, &msgs); err != nil {
		return nil, err
	}
	return msgs, nil
}

type discordChannel struct {
	ID   string `json:"id"`
	Type int    `json:"type"`
}

type discordMessage struct {
	ID        string           `json:"id"`
	ChannelID string           `json:"channel_id"`
	Content   string           `json:"content"`
	Timestamp string           `json:"timestamp"`
	Author    *discordUser     `json:"author"`
	Mentions  []discordUser    `json:"mentions"`
	Embeds    []discordEmbed   `json:"embeds"`
	Thread    *discordChannel  `json:"thread"`
}

type discordUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type discordEmbed struct {
	Description string `json:"description"`
}

type discordPollMsg struct {
	channelID string
	msg       discordMessage
}

func parseDiscordTimestamp(ts string) int64 {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, ts)
	}
	return t.Unix()
}

func parseDiscordCursor(s string) map[string]string {
	if s == "" {
		return nil
	}
	var m map[string]string
	_ = json.Unmarshal([]byte(s), &m)
	return m
}

func encodeDiscordCursor(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	b, _ := json.Marshal(m)
	return string(b)
}
