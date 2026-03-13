package inference

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	chatJimmyDefaultBaseURL = "https://chatjimmy.ai"
	chatJimmyDefaultModel   = "llama3.1-8B"
	chatJimmyUserAgent      = "mormoneyos/1.0"
)

// chatJimmyStatsRe matches <|stats|>{...}<|/stats|> per chatjimmy-cli API reference.
var chatJimmyStatsRe = regexp.MustCompile(`<\|stats\|>([\s\S]+?)<\|/stats\|>`)

// ChatJimmyClient implements inference.Client for chatjimmy.ai (Taalas HC1 inference, no auth).
// API reference: https://github.com/kichichifightclubx/chatjimmy-cli/blob/main/docs/02-api-reference.md
type ChatJimmyClient struct {
	BaseURL   string
	Model     string
	MaxTokens int
	HTTP      *http.Client
	lowCompute bool
}

// NewChatJimmyClient creates a client for chatjimmy.ai.
func NewChatJimmyClient(baseURL, model string, maxTokens int) *ChatJimmyClient {
	if baseURL == "" {
		baseURL = chatJimmyDefaultBaseURL
	}
	baseURL = strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	if model == "" {
		model = chatJimmyDefaultModel
	}
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	return &ChatJimmyClient{
		BaseURL:   baseURL,
		Model:     model,
		MaxTokens: maxTokens,
		HTTP: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

// Chat implements inference.Client.
func (c *ChatJimmyClient) Chat(ctx context.Context, messages []ChatMessage, opts *InferenceOptions) (*InferenceResponse, error) {
	model := c.Model
	if opts != nil && opts.Model != "" {
		model = opts.Model
	}

	systemPrompt := ""
	var chatMessages []chatJimmyMessage
	for _, m := range messages {
		if m.Role == "system" {
			systemPrompt = m.Content
			continue
		}
		if m.Role == "user" || m.Role == "assistant" {
			chatMessages = append(chatMessages, chatJimmyMessage{Role: m.Role, Content: m.Content})
		}
	}

	if len(chatMessages) == 0 {
		chatMessages = []chatJimmyMessage{{Role: "user", Content: ""}}
	}

	reqBody := chatJimmyRequest{
		Messages: chatMessages,
		ChatOptions: chatJimmyOptions{
			SelectedModel: model,
			SystemPrompt:  systemPrompt,
			TopK:          8,
		},
		Attachment: nil,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("chatjimmy marshal: %w", err)
	}

	url := c.BaseURL + "/api/chat"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("chatjimmy request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", chatJimmyUserAgent)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chatjimmy: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("chatjimmy read: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chatjimmy API: %d %s", resp.StatusCode, sanitizeAPIError(string(b)))
	}

	full := string(b)
	content, inputTokens, outputTokens := parseChatJimmyResponse(full)
	if content == "" {
		return nil, fmt.Errorf("chatjimmy returned empty response (input may exceed prefill limit ~6k tokens)")
	}

	return &InferenceResponse{
		Content:       content,
		InputTokens:   inputTokens,
		OutputTokens:  outputTokens,
		FinishReason:  "stop",
		CostCents:     0,
	}, nil
}

// GetDefaultModel implements inference.Client.
func (c *ChatJimmyClient) GetDefaultModel() string {
	return c.Model
}

// SetLowComputeMode implements inference.Client.
func (c *ChatJimmyClient) SetLowComputeMode(enabled bool) {
	c.lowCompute = enabled
}

// Health checks the ChatJimmy API (GET /api/health). Returns true when status and backend are ok.
func (c *ChatJimmyClient) Health(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/health", nil)
	if err != nil {
		return false, fmt.Errorf("chatjimmy health request: %w", err)
	}
	req.Header.Set("User-Agent", chatJimmyUserAgent)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return false, fmt.Errorf("chatjimmy health: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("chatjimmy health: %d %s", resp.StatusCode, sanitizeAPIError(string(b)))
	}

	var h chatJimmyHealthStatus
	if err := json.NewDecoder(resp.Body).Decode(&h); err != nil {
		return false, fmt.Errorf("chatjimmy health decode: %w", err)
	}
	return h.Status == "ok" && h.Backend == "healthy", nil
}

// Models returns available models from GET /api/models.
func (c *ChatJimmyClient) Models(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/models", nil)
	if err != nil {
		return nil, fmt.Errorf("chatjimmy models request: %w", err)
	}
	req.Header.Set("User-Agent", chatJimmyUserAgent)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chatjimmy models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("chatjimmy models: %d %s", resp.StatusCode, sanitizeAPIError(string(b)))
	}

	var mr chatJimmyModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&mr); err != nil {
		return nil, fmt.Errorf("chatjimmy models decode: %w", err)
	}
	ids := make([]string, 0, len(mr.Data))
	for _, m := range mr.Data {
		ids = append(ids, m.ID)
	}
	return ids, nil
}

// chatJimmyHealthStatus matches GET /api/health response (chatjimmy-cli API reference §2.2).
type chatJimmyHealthStatus struct {
	Status string `json:"status"`
	Backend string `json:"backend"`
}

// chatJimmyModelsResponse matches GET /api/models response (chatjimmy-cli API reference §2.3).
type chatJimmyModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// parseChatJimmyResponse extracts text and token counts from API response.
// Per chatjimmy-cli: text is everything before <|stats|>{...}<|/stats|>; stats hold prefill/decode tokens.
func parseChatJimmyResponse(full string) (text string, inputTokens, outputTokens int) {
	full = strings.TrimSpace(full)
	if m := chatJimmyStatsRe.FindStringSubmatch(full); len(m) > 1 {
		idx := strings.Index(full, "<|stats|>")
		text = strings.TrimSpace(full[:idx])
		var stats chatJimmyStats
		if err := json.Unmarshal([]byte(m[1]), &stats); err == nil {
			return text, stats.PrefillTokens, stats.DecodeTokens
		}
		return text, 0, 0
	}
	return full, 0, 0
}

// chatJimmyStats matches chatjimmy-cli internal/client/types.go Stats (API reference §4).
type chatJimmyStats struct {
	PrefillTokens int `json:"prefill_tokens"`
	DecodeTokens  int `json:"decode_tokens"`
}

func sanitizeAPIError(s string) string {
	const max = 200
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

type chatJimmyMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatJimmyOptions struct {
	SelectedModel string `json:"selectedModel"`
	SystemPrompt  string `json:"systemPrompt"`
	TopK          int    `json:"topK"`
}

type chatJimmyRequest struct {
	Messages    []chatJimmyMessage `json:"messages"`
	ChatOptions chatJimmyOptions  `json:"chatOptions"`
	Attachment  *struct{}         `json:"attachment,omitempty"`
}

var _ Client = (*ChatJimmyClient)(nil)
