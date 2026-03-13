package inference

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AuthStyle controls how the API key is sent in requests.
type AuthStyle int

const (
	AuthBearer AuthStyle = iota
	AuthXApiKey
)

// OpenAICompatibleClient is a single implementation for all /v1/chat/completions APIs.
// Used by OpenAI, Conway, Ollama, Groq, Mistral, DeepSeek, etc.
type OpenAICompatibleClient struct {
	Name      string
	BaseURL   string
	APIKey    string
	AuthStyle AuthStyle
	Model     string
	MaxTokens int
	HTTP      *http.Client
	lowCompute bool
}

// NewOpenAICompatibleClient creates a client for any OpenAI-compatible endpoint.
func NewOpenAICompatibleClient(name, baseURL, apiKey string, auth AuthStyle, model string, maxTokens int) *OpenAICompatibleClient {
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	return &OpenAICompatibleClient{
		Name:      name,
		BaseURL:   baseURL,
		APIKey:    apiKey,
		AuthStyle: auth,
		Model:     model,
		MaxTokens: maxTokens,
		HTTP: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

// Chat implements inference.Client.
func (c *OpenAICompatibleClient) Chat(ctx context.Context, messages []ChatMessage, opts *InferenceOptions) (*InferenceResponse, error) {
	model := c.Model
	maxTokens := c.MaxTokens
	if opts != nil {
		if opts.Model != "" {
			model = opts.Model
		}
		if opts.MaxTokens > 0 {
			maxTokens = opts.MaxTokens
		}
	}

	reqBody := map[string]any{
		"model":      model,
		"messages":   formatMessages(messages),
		"max_tokens": maxTokens,
		"stream":     false,
	}
	if opts != nil && opts.Temperature > 0 {
		reqBody["temperature"] = opts.Temperature
	}
	if opts != nil && len(opts.Tools) > 0 {
		reqBody["tools"] = opts.Tools
		reqBody["tool_choice"] = "auto"
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.BaseURL
	if len(url) > 0 && url[len(url)-1] == '/' {
		url = url[:len(url)-1]
	}
	url += "/v1/chat/completions"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		switch c.AuthStyle {
		case AuthBearer:
			req.Header.Set("Authorization", "Bearer "+c.APIKey)
		case AuthXApiKey:
			req.Header.Set("x-api-key", c.APIKey)
		default:
			req.Header.Set("Authorization", "Bearer "+c.APIKey)
		}
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("inference %s: %d %s", url, resp.StatusCode, string(b))
	}

	var data openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return parseOpenAIResponse(&data, model)
}

// GetDefaultModel implements inference.Client.
func (c *OpenAICompatibleClient) GetDefaultModel() string {
	return c.Model
}

// SetLowComputeMode implements inference.Client.
func (c *OpenAICompatibleClient) SetLowComputeMode(enabled bool) {
	c.lowCompute = enabled
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens    int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens     int `json:"total_tokens"`
	} `json:"usage"`
}

func parseOpenAIResponse(data *openAIResponse, model string) (*InferenceResponse, error) {
	if len(data.Choices) == 0 {
		return &InferenceResponse{
			Content:      "",
			InputTokens:  data.Usage.PromptTokens,
			OutputTokens: data.Usage.CompletionTokens,
			FinishReason: "stop",
			CostCents:    0,
		}, nil
	}
	choice := data.Choices[0]
	msg := choice.Message

	out := &InferenceResponse{
		Content:       msg.Content,
		InputTokens:   data.Usage.PromptTokens,
		OutputTokens:  data.Usage.CompletionTokens,
		FinishReason:  choice.FinishReason,
		CostCents:     0,
	}
	for _, tc := range msg.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}
	return out, nil
}

func formatMessages(msgs []ChatMessage) []map[string]any {
	out := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		obj := map[string]any{"role": m.Role}
		if m.Content != "" {
			obj["content"] = m.Content
		}
		if len(m.ToolCalls) > 0 {
			var tcs []map[string]any
			for _, tc := range m.ToolCalls {
				tcs = append(tcs, map[string]any{
					"id":   tc.ID,
					"type": tc.Type,
					"function": map[string]any{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				})
			}
			obj["tool_calls"] = tcs
		}
		out = append(out, obj)
	}
	return out
}

// Ensure OpenAICompatibleClient implements Client.
var _ Client = (*OpenAICompatibleClient)(nil)
