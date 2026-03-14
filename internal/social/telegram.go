package social

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/morpheumlabs/mormoneyos-go/internal/config"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

const telegramAPIBase = "https://api.telegram.org/bot"

// TelegramChannel implements SocialChannel for Telegram Bot API.
type TelegramChannel struct {
	tokenMgr     *TokenManager
	allowedUsers map[string]struct{} // username or user_id string; empty = allow all
}

// NewTelegramChannel creates a Telegram channel. Requires TelegramBotToken.
func NewTelegramChannel(cfg *types.AutomatonConfig) (SocialChannel, error) {
	if cfg == nil || cfg.TelegramBotToken == "" {
		return nil, fmt.Errorf("telegram: bot token required")
	}
	ch := &TelegramChannel{
		allowedUsers: make(map[string]struct{}),
	}
	for _, u := range cfg.TelegramAllowedUsers {
		u = strings.TrimSpace(strings.ToLower(u))
		if u != "" {
			ch.allowedUsers[u] = struct{}{}
		}
	}
	ch.tokenMgr = NewTokenManager(
		func(ctx context.Context) (string, time.Time, error) {
			// Re-read config so token updates via API apply without restart
			c, err := config.Load()
			if err != nil {
				return "", time.Time{}, err
			}
			if c == nil || c.TelegramBotToken == "" {
				return "", time.Time{}, fmt.Errorf("telegram: bot token not in config")
			}
			return c.TelegramBotToken, time.Now().Add(365 * 24 * time.Hour), nil
		},
		slog.With("channel", "telegram"),
	)
	return ch, nil
}

func (c *TelegramChannel) Name() string {
	return "telegram"
}

// GetAuthToken returns the bot token (refreshed from config if invalidated).
func (c *TelegramChannel) GetAuthToken(ctx context.Context) (string, error) {
	return c.tokenMgr.GetAuthToken(ctx)
}

// Invalidate clears cached token so next GetAuthToken re-reads from config.
func (c *TelegramChannel) Invalidate() {
	c.tokenMgr.Invalidate()
}

// doRequest performs an HTTP request with auth. On 401/403, invalidates token for next call.
func (c *TelegramChannel) doRequest(ctx context.Context, method, url string, body []byte) (*http.Response, error) {
	token, err := c.tokenMgr.GetAuthToken(ctx)
	if err != nil {
		return nil, err
	}
	fullURL := telegramAPIBase + token + url
	var req *http.Request
	if len(body) > 0 {
		req, err = http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(body))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, fullURL, nil)
	}
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		c.tokenMgr.Invalidate()
	}
	return resp, nil
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
	resp, err := c.doRequest(ctx, http.MethodPost, "/sendMessage", body)
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

	path := fmt.Sprintf("/getUpdates?offset=%d&limit=%d", offset, limit)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
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
	resp, err := c.doRequest(ctx, http.MethodGet, "/getMe", nil)
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
