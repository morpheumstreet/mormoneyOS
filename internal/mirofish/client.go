package mirofish

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// Client provides MiroFish API operations (swarm intelligence / foresight layer).
type Client interface {
	// Call invokes an MiroFish API endpoint (simulate, report, inject, chat, heartbeat).
	Call(ctx context.Context, path string, payload map[string]any) ([]byte, int, error)
}

// HTTPClient is a resilient HTTP client for MiroFish API.
type HTTPClient struct {
	BaseURL string
	HTTP    *http.Client
	cfg     *types.MiroFishConfig
}

// NewHTTPClient creates a MiroFish HTTP client from config.
func NewHTTPClient(cfg *types.MiroFishConfig) *HTTPClient {
	if cfg == nil {
		cfg = &types.MiroFishConfig{
			Enabled:        true,
			BaseURL:        "http://localhost:5001",
			TimeoutSeconds: 300,
			DefaultLLM:    "qwen-plus",
			MaxAgents:      2000,
		}
	}
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 300 * time.Second
	}
	return &HTTPClient{
		BaseURL: strings.TrimSuffix(cfg.BaseURL, "/"),
		HTTP:    &http.Client{Timeout: timeout},
		cfg:     cfg,
	}
}

// Call invokes an MiroFish API endpoint.
func (c *HTTPClient) Call(ctx context.Context, path string, payload map[string]any) ([]byte, int, error) {
	url := c.BaseURL + "/api/" + path
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("MiroFish request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}
	return respBody, resp.StatusCode, nil
}

// Ensure HTTPClient implements Client.
var _ Client = (*HTTPClient)(nil)
