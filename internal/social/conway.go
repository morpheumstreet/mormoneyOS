package social

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/morpheumlabs/mormoneyos-go/internal/identity"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

const requestTimeout = 30 * time.Second

// ConwayChannel implements SocialChannel for the Conway social relay (wallet-signed agent-to-agent).
type ConwayChannel struct {
	baseURL string
	account *identity.EVMAccount
}

// NewConwayChannel creates a Conway channel. Requires socialRelayURL and wallet.
func NewConwayChannel(cfg *types.AutomatonConfig) (SocialChannel, error) {
	url := cfg.SocialRelayURL
	if url == "" {
		return nil, fmt.Errorf("conway: socialRelayURL required")
	}
	url = strings.TrimSuffix(url, "/")
	if !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("conway: relay URL must use HTTPS")
	}
	account, _, err := identity.GetWallet()
	if err != nil {
		return nil, fmt.Errorf("conway: wallet required for signing: %w", err)
	}
	return &ConwayChannel{baseURL: url, account: account}, nil
}

func (c *ConwayChannel) Name() string {
	return "conway"
}

func (c *ConwayChannel) Send(ctx context.Context, msg *OutboundMessage) (string, error) {
	if err := ValidateOutbound(msg.Content); err != nil {
		return "", err
	}
	to := strings.ToLower(strings.TrimSpace(msg.Recipient))
	if !isValidEthAddress(to) {
		return "", fmt.Errorf("conway: invalid recipient address %q", msg.Recipient)
	}

	signedAt := time.Now().UTC().Format(time.RFC3339)
	contentHash := crypto.Keccak256Hash([]byte(msg.Content)).Hex()
	canonical := fmt.Sprintf("Conway:send:%s:%s:%s", to, contentHash, signedAt)

	sig, err := c.account.SignMessage([]byte(canonical))
	if err != nil {
		return "", fmt.Errorf("conway: sign: %w", err)
	}
	sigHex := "0x" + hex.EncodeToString(sig)

	payload := map[string]any{
		"from":       strings.ToLower(c.account.Address()),
		"to":         to,
		"content":    msg.Content,
		"signed_at":  signedAt,
		"signature":  sigHex,
		"reply_to":   msg.ReplyTo,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("conway send: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(respBody, &errResp)
		if errResp.Error != "" {
			return "", fmt.Errorf("conway send %d: %s", resp.StatusCode, errResp.Error)
		}
		return "", fmt.Errorf("conway send %d: %s", resp.StatusCode, string(respBody))
	}

	var data struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &data); err != nil {
		return "", fmt.Errorf("conway: parse response: %w", err)
	}
	return data.ID, nil
}

func (c *ConwayChannel) Poll(ctx context.Context, cursor string, limit int) ([]InboxMessage, string, error) {
	if limit <= 0 {
		limit = 50
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	addr := strings.ToLower(c.account.Address())
	canonical := fmt.Sprintf("Conway:poll:%s:%s", addr, timestamp)

	sig, err := c.account.SignMessage([]byte(canonical))
	if err != nil {
		return nil, "", fmt.Errorf("conway: sign poll: %w", err)
	}
	sigHex := "0x" + hex.EncodeToString(sig)

	body, _ := json.Marshal(map[string]any{"cursor": cursor, "limit": limit})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages/poll", bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Wallet-Address", addr)
	req.Header.Set("X-Signature", sigHex)
	req.Header.Set("X-Timestamp", timestamp)

	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("conway poll: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(respBody, &errResp)
		if errResp.Error != "" {
			return nil, "", fmt.Errorf("conway poll %d: %s", resp.StatusCode, errResp.Error)
		}
		return nil, "", fmt.Errorf("conway poll %d: %s", resp.StatusCode, string(respBody))
	}

	var data struct {
		Messages   []struct {
			ID        string `json:"id"`
			From      string `json:"from"`
			To        string `json:"to"`
			Content   string `json:"content"`
			SignedAt  string `json:"signedAt"`
			CreatedAt string `json:"createdAt"`
			ReplyTo   string `json:"replyTo"`
		} `json:"messages"`
		NextCursor string `json:"next_cursor"`
	}
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, "", fmt.Errorf("conway: parse poll response: %w", err)
	}

	out := make([]InboxMessage, 0, len(data.Messages))
	for _, m := range data.Messages {
		ts, _ := time.Parse(time.RFC3339, m.CreatedAt)
		out = append(out, InboxMessage{
			ID:          m.ID,
			Sender:      m.From,
			ReplyTarget: m.From, // reply to sender
			Content:     m.Content,
			Channel:     "conway",
			Timestamp:   ts.Unix(),
			ThreadID:    m.ReplyTo,
		})
	}
	return out, data.NextCursor, nil
}

func (c *ConwayChannel) HealthCheck(ctx context.Context) error {
	// Poll with limit=0 to verify relay is reachable
	_, _, err := c.Poll(ctx, "", 1)
	return err
}

// GetAuthToken returns a sentinel for Conway (wallet-signed per-request; no token).
func (c *ConwayChannel) GetAuthToken(ctx context.Context) (string, error) {
	return "conway-signed", nil
}

// Invalidate is a no-op for Conway (wallet signing never expires).
func (c *ConwayChannel) Invalidate() {}

func isValidEthAddress(s string) bool {
	if len(s) != 42 || !strings.HasPrefix(s, "0x") {
		return false
	}
	for _, c := range s[2:] {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			continue
		}
		return false
	}
	return true
}
