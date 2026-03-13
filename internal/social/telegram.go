package social

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

const telegramAPIBase = "https://api.telegram.org/bot"

// TelegramChannel implements SocialChannel for Telegram Bot API.
type TelegramChannel struct {
	token        string
	allowedUsers map[string]struct{} // username or user_id string; empty = allow all
}

// NewTelegramChannel creates a Telegram channel. Requires TelegramBotToken.
func NewTelegramChannel(cfg *types.AutomatonConfig) (SocialChannel, error) {
	if cfg == nil || cfg.TelegramBotToken == "" {
		return nil, fmt.Errorf("telegram: bot token required")
	}
	ch := &TelegramChannel{
		token:        cfg.TelegramBotToken,
		allowedUsers: make(map[string]struct{}),
	}
	for _, u := range cfg.TelegramAllowedUsers {
		u = strings.TrimSpace(strings.ToLower(u))
		if u != "" {
			ch.allowedUsers[u] = struct{}{}
		}
	}
	return ch, nil
}

func (c *TelegramChannel) Name() string {
	return "telegram"
}

func (c *TelegramChannel) Send(ctx context.Context, msg *OutboundMessage) (string, error) {
	if err := ValidateOutbound(msg.Content); err != nil {
		return "", err
	}
	chatID := strings.TrimSpace(msg.Recipient)
	if chatID == "" {
		return "", fmt.Errorf("telegram: recipient (chat_id) required")
	}

	payload := map[string]any{
		"chat_id": chatID,
		"text":    msg.Content,
	}
	if msg.ReplyTo != "" {
		if mid, err := strconv.ParseInt(msg.ReplyTo, 10, 64); err == nil {
			payload["reply_to_message_id"] = mid
		}
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, telegramAPIBase+c.token+"/sendMessage", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("telegram send: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		OK     bool `json:"ok"`
		Result *struct {
			MessageID int64 `json:"message_id"`
		} `json:"result"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("telegram: parse response: %w", err)
	}
	if !result.OK {
		if result.Description != "" {
			return "", fmt.Errorf("telegram send: %s", result.Description)
		}
		return "", fmt.Errorf("telegram send: %s", string(respBody))
	}
	if result.Result == nil {
		return "", fmt.Errorf("telegram: no message_id in response")
	}
	return strconv.FormatInt(result.Result.MessageID, 10), nil
}

func (c *TelegramChannel) Poll(ctx context.Context, cursor string, limit int) ([]InboxMessage, string, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	offset := 0
	if cursor != "" {
		if o, err := strconv.ParseInt(cursor, 10, 64); err == nil {
			offset = int(o)
		}
	}

	url := fmt.Sprintf("%s%s/getUpdates?offset=%d&limit=%d", telegramAPIBase, c.token, offset, limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}

	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("telegram poll: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		OK     bool `json:"ok"`
		Result []struct {
			UpdateID int64 `json:"update_id"`
			Message  *struct {
				MessageID int64 `json:"message_id"`
				From      *struct {
					ID        int64  `json:"id"`
					Username  string `json:"username"`
					FirstName string `json:"first_name"`
				} `json:"from"`
				Chat struct {
					ID    int64  `json:"id"`
					Type  string `json:"type"`
					Title string `json:"title"`
				} `json:"chat"`
				Text string `json:"text"`
				Date int64  `json:"date"`
			} `json:"message"`
		} `json:"result"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, "", fmt.Errorf("telegram: parse poll response: %w", err)
	}
	if !result.OK {
		if result.Description != "" {
			return nil, "", fmt.Errorf("telegram poll: %s", result.Description)
		}
		return nil, "", fmt.Errorf("telegram poll: %s", string(respBody))
	}

	out := make([]InboxMessage, 0, len(result.Result))
	nextOffset := offset
	for _, u := range result.Result {
		if u.UpdateID >= int64(nextOffset) {
			nextOffset = int(u.UpdateID) + 1
		}
		if u.Message == nil {
			continue
		}
		msg := u.Message
		if msg.Text == "" {
			continue
		}
		senderID := ""
		senderKey := ""
		if msg.From != nil {
			senderID = strconv.FormatInt(msg.From.ID, 10)
			if msg.From.Username != "" {
				senderKey = strings.ToLower("@" + msg.From.Username)
			} else {
				senderKey = senderID
			}
		}
		if len(c.allowedUsers) > 0 {
			if _, ok := c.allowedUsers[senderKey]; !ok {
				if _, ok := c.allowedUsers[senderID]; !ok {
					continue
				}
			}
		}
		out = append(out, InboxMessage{
			ID:          strconv.FormatInt(msg.MessageID, 10),
			Sender:      senderID,
			ReplyTarget: strconv.FormatInt(msg.Chat.ID, 10),
			Content:     msg.Text,
			Channel:     "telegram",
			Timestamp:   msg.Date,
			ThreadID:    "",
		})
	}
	return out, strconv.Itoa(nextOffset), nil
}

func (c *TelegramChannel) HealthCheck(ctx context.Context) error {
	url := telegramAPIBase + c.token + "/getMe"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram health: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("telegram health: parse: %w", err)
	}
	if !result.OK {
		if result.Description != "" {
			return fmt.Errorf("telegram health: %s", result.Description)
		}
		return fmt.Errorf("telegram health: %s", string(respBody))
	}
	return nil
}
