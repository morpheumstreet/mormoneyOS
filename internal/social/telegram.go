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
	"sync"
	"time"
	"unicode/utf16"

	"github.com/morpheumlabs/mormoneyos-go/internal/config"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

const telegramAPIBase = "https://api.telegram.org/bot"

// TooManyRequestsError is returned when Telegram API returns 429 (flood control).
// RetryAfter is the seconds to wait before retrying.
type TooManyRequestsError struct {
	RetryAfter int
}

func (e *TooManyRequestsError) Error() string {
	return fmt.Sprintf("telegram: too many requests, retry after %d seconds", e.RetryAfter)
}

// ConflictError is returned when Telegram API returns 409 (another getUpdates in progress).
// Call deleteWebhook and retry after short backoff.
type ConflictError struct{}

func (e *ConflictError) Error() string {
	return "telegram: 409 conflict (another getUpdates in progress)"
}

// TelegramChannel implements SocialChannel for Telegram Bot API.
// OpenClaw-style: allowFrom (empty=deny, ["*"]=allow), groups allowlist, requireMention in groups.
type TelegramChannel struct {
	tokenMgr *TokenManager

	// DM allowlist: allowAll=true when ["*"]; allowSet populated for specific users; else deny all
	allowAll bool
	allowSet map[string]struct{}

	// Group allowlist: allowAllGroups when ["*"]; groupSet for specific chat IDs
	allowAllGroups bool
	groupSet      map[string]struct{}

	// requireMention: in groups, only respond when bot is @mentioned
	requireMentionDefault bool
	groupConfig          map[string]types.TelegramGroupCfg

	// Bot username for mention detection (e.g. "mybot")
	botUsername   string
	botUsernameMu sync.RWMutex
}

// NewTelegramChannel creates a Telegram channel. Requires TelegramBotToken.
func NewTelegramChannel(cfg *types.AutomatonConfig) (SocialChannel, error) {
	if cfg == nil || cfg.TelegramBotToken == "" {
		return nil, fmt.Errorf("telegram: bot token required")
	}
	ch := &TelegramChannel{
		allowSet:              make(map[string]struct{}),
		groupSet:              make(map[string]struct{}),
		groupConfig:           make(map[string]types.TelegramGroupCfg),
		requireMentionDefault: true, // safe default for groups
	}
	// Allowlist: empty = deny all; ["*"] = allow all; else exact match
	for _, u := range cfg.TelegramAllowedUsers {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if strings.EqualFold(u, "*") {
			ch.allowAll = true
			break
		}
		ch.allowSet[strings.ToLower(u)] = struct{}{}
	}
	// Groups: ["*"] = all; else specific chat IDs
	for _, g := range cfg.TelegramGroups {
		g = strings.TrimSpace(g)
		if g == "" {
			continue
		}
		if strings.EqualFold(g, "*") {
			ch.allowAllGroups = true
			break
		}
		ch.groupSet[g] = struct{}{}
	}
	if cfg.TelegramRequireMention != nil {
		ch.requireMentionDefault = *cfg.TelegramRequireMention
	}
	for k, gc := range cfg.TelegramGroupsConfig {
		ch.groupConfig[k] = gc
	}
	ch.tokenMgr = NewTokenManager(
		func(ctx context.Context) (string, time.Time, error) {
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

func (c *TelegramChannel) GetAuthToken(ctx context.Context) (string, error) {
	return c.tokenMgr.GetAuthToken(ctx)
}

func (c *TelegramChannel) Invalidate() {
	c.tokenMgr.Invalidate()
	c.botUsernameMu.Lock()
	c.botUsername = ""
	c.botUsernameMu.Unlock()
}

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
	if resp.StatusCode == 429 {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var errResp struct {
			Parameters struct {
				RetryAfter int `json:"retry_after"`
			} `json:"parameters"`
		}
		_ = json.Unmarshal(respBody, &errResp)
		retrySec := errResp.Parameters.RetryAfter
		if retrySec <= 0 {
			retrySec = 60
		}
		return nil, &TooManyRequestsError{RetryAfter: retrySec}
	}
	if resp.StatusCode == 409 {
		resp.Body.Close()
		return nil, &ConflictError{}
	}
	return resp, nil
}

// ensureBotUsername fetches bot username from getMe if not cached.
func (c *TelegramChannel) ensureBotUsername(ctx context.Context) (string, error) {
	c.botUsernameMu.RLock()
	u := c.botUsername
	c.botUsernameMu.RUnlock()
	if u != "" {
		return u, nil
	}
	resp, err := c.doRequest(ctx, http.MethodGet, "/getMe", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		OK     bool `json:"ok"`
		Result *struct {
			Username string `json:"username"`
		} `json:"result"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil || !result.OK || result.Result == nil {
		return "", nil
	}
	u = strings.ToLower(result.Result.Username)
	c.botUsernameMu.Lock()
	c.botUsername = u
	c.botUsernameMu.Unlock()
	return u, nil
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
	if msg.ParseMode != "" {
		payload["parse_mode"] = msg.ParseMode
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

// telegramMessage is the raw message from getUpdates (with entities and caption).
type telegramMessage struct {
	MessageID int64 `json:"message_id"`
	From      *struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		FirstName string `json:"first_name"`
	} `json:"from"`
	Chat struct {
		ID    int64  `json:"id"`
		Type  string `json:"type"` // "private", "group", "supergroup", "channel"
		Title string `json:"title"`
	} `json:"chat"`
	Text     string             `json:"text"`
	Caption  string             `json:"caption"`
	Date     int64              `json:"date"`
	Entities []telegramEntity   `json:"entities"`
}

type telegramEntity struct {
	Type   string `json:"type"` // "mention", "text_mention", "bot_command"
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	User   *struct {
		Username string `json:"username"`
	} `json:"user"`
}

// entitySlice extracts substring from text using Telegram's UTF-16 offset/length (core.telegram.org/api/entities).
func entitySlice(text string, offset, length int) string {
	runes := []rune(text)
	utf16Units := utf16.Encode(runes)
	if offset < 0 || length <= 0 || offset >= len(utf16Units) {
		return ""
	}
	end := offset + length
	if end > len(utf16Units) {
		end = len(utf16Units)
	}
	decoded := utf16.Decode(utf16Units[offset:end])
	return string(decoded)
}

// isBotMentioned returns true if the bot is @mentioned in text using entities.
// Entity offset/length are UTF-16 code units per Telegram API, not bytes.
func isBotMentioned(text string, entities []telegramEntity, botUsername string) bool {
	if botUsername == "" {
		return false
	}
	botMention := "@" + botUsername
	for _, e := range entities {
		if e.Type != "mention" && e.Type != "text_mention" && e.Type != "bot_command" {
			continue
		}
		slice := entitySlice(text, e.Offset, e.Length)
		if slice == "" {
			continue
		}
		if strings.EqualFold(slice, botMention) || strings.EqualFold(slice, "/"+botUsername) {
			return true
		}
		if e.Type == "text_mention" && e.User != nil && strings.EqualFold(e.User.Username, botUsername) {
			return true
		}
	}
	return false
}

func (c *TelegramChannel) Poll(ctx context.Context, cursor string, limit int) ([]InboxMessage, string, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	var offset int64
	if cursor != "" {
		if o, err := strconv.ParseInt(cursor, 10, 64); err == nil {
			offset = o
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
			UpdateID int64            `json:"update_id"`
			Message  *telegramMessage  `json:"message"`
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

	botUsername, _ := c.ensureBotUsername(ctx)

	out := make([]InboxMessage, 0, len(result.Result))
	var maxUpdateID int64
	for _, u := range result.Result {
		if u.UpdateID > maxUpdateID {
			maxUpdateID = u.UpdateID
		}
		if u.Message == nil {
			continue
		}
		msg := u.Message
		content := msg.Text
		if content == "" {
			content = msg.Caption
		}
		if content == "" {
			continue
		}

		chatIDStr := strconv.FormatInt(msg.Chat.ID, 10)
		isGroup := msg.Chat.Type == "group" || msg.Chat.Type == "supergroup"

		// Group filter: if groups configured, only allow listed groups
		if isGroup {
			if !c.allowAllGroups {
				if len(c.groupSet) == 0 {
					continue
				}
				if _, ok := c.groupSet[chatIDStr]; !ok {
					continue
				}
			}
			// requireMention: only process when bot is @mentioned
			reqMention := c.requireMentionDefault
			if gc, ok := c.groupConfig[chatIDStr]; ok {
				reqMention = gc.RequireMention
			}
			if reqMention && !isBotMentioned(content, msg.Entities, botUsername) {
				continue
			}
		} else {
			// DM: apply allowlist (empty = deny all, ["*"] = allow all)
			if !c.allowAll {
				if len(c.allowSet) == 0 {
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
				if _, ok := c.allowSet[senderKey]; !ok {
					if _, ok := c.allowSet[senderID]; !ok {
						continue
					}
				}
			}
		}

		senderID := ""
		if msg.From != nil {
			senderID = strconv.FormatInt(msg.From.ID, 10)
		}
		out = append(out, InboxMessage{
			ID:          strconv.FormatInt(msg.MessageID, 10),
			Sender:      senderID,
			ReplyTarget: chatIDStr,
			Content:     content,
			Channel:     "telegram",
			Timestamp:   msg.Date,
			ThreadID:    "",
		})
	}
	// Advance offset past all received updates so Telegram confirms them (avoids stuck/repeated messages)
	var nextOffset int64
	if maxUpdateID > 0 {
		nextOffset = maxUpdateID + 1
	} else {
		nextOffset = offset
	}
	return out, strconv.FormatInt(nextOffset, 10), nil
}

// DeleteWebhook removes any webhook so getUpdates (long-polling) works.
// OpenClaw-aligned: call before polling to avoid 409 conflicts.
func (c *TelegramChannel) DeleteWebhook(ctx context.Context) error {
	resp, err := c.doRequest(ctx, http.MethodGet, "/deleteWebhook", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Best-effort: even if deleteWebhook fails (e.g. no webhook), polling can proceed
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	_ = json.Unmarshal(body, &result)
	if !result.OK && result.Description != "" {
		slog.Default().Debug("telegram deleteWebhook", "description", result.Description)
	}
	return nil
}

func (c *TelegramChannel) HealthCheck(ctx context.Context) error {
	// Clear any leftover webhook before polling (OpenClaw-aligned)
	_ = c.DeleteWebhook(ctx)
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
	// Cache bot username for mention detection
	_, _ = c.ensureBotUsername(ctx)
	// Register commands so they appear in the chat box when user types "/"
	if err := c.setMyCommands(ctx); err != nil {
		slog.Default().Warn("telegram setMyCommands failed", "err", err)
		// Non-fatal: bot still works, commands just won't show in UI
	}
	return nil
}

// telegramBotCommand matches Telegram Bot API BotCommand.
type telegramBotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// setMyCommands registers bot commands with Telegram so they appear in the chat input menu.
func (c *TelegramChannel) setMyCommands(ctx context.Context) error {
	commands := []telegramBotCommand{
		{Command: "ping", Description: "instant pong"},
		{Command: "status", Description: "agent state, turns, credits, tier"},
		{Command: "help", Description: "list commands"},
		{Command: "balance", Description: "economic status, USDC by wallet"},
		{Command: "skill", Description: "list all skills"},
		{Command: "pause", Description: "pause agent"},
		{Command: "resume", Description: "resume agent"},
		{Command: "reset", Description: "request context reset (wake agent)"},
	}
	payload := map[string]any{"commands": commands}
	body, _ := json.Marshal(payload)
	resp, err := c.doRequest(ctx, http.MethodPost, "/setMyCommands", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse setMyCommands response: %w", err)
	}
	if !result.OK {
		if result.Description != "" {
			return fmt.Errorf("setMyCommands: %s", result.Description)
		}
		return fmt.Errorf("setMyCommands: %s", string(respBody))
	}
	return nil
}
