package inference

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// LatencyProber measures response time for local inference providers.
// Implementations must be safe for concurrent use.
type LatencyProber interface {
	// Probe sends a minimal completion request and returns latency in milliseconds.
	// provider must be a local provider (ollama, localai, llamacpp, lmstudio, vllm, janai, g4f).
	Probe(ctx context.Context, provider, baseURL, modelID string) (latencyMs int64, err error)
}

// DefaultLatencyProber is the standard implementation that uses OpenAI-compatible
// or provider-specific endpoints to measure response time.
type DefaultLatencyProber struct {
	HTTPClient *http.Client
}

// NewLatencyProber creates a DefaultLatencyProber with a 30s timeout.
func NewLatencyProber() *DefaultLatencyProber {
	return &DefaultLatencyProber{
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Probe implements LatencyProber.
func (p *DefaultLatencyProber) Probe(ctx context.Context, provider, baseURL, modelID string) (int64, error) {
	provider = strings.TrimSpace(strings.ToLower(provider))
	baseURL = strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || modelID == "" {
		return 0, fmt.Errorf("baseURL and modelID required")
	}
	spec := LookupProvider(provider)
	if spec == nil || !spec.Local {
		return 0, fmt.Errorf("provider must be a local provider (ollama, localai, llamacpp, lmstudio, vllm, janai, g4f)")
	}

	start := time.Now()
	var err error
	switch provider {
	case "ollama":
		err = p.probeOllama(ctx, baseURL, modelID)
	default:
		err = p.probeOpenAICompatible(ctx, baseURL, modelID)
	}
	elapsed := time.Since(start)
	if err != nil {
		return 0, err
	}
	return elapsed.Milliseconds(), nil
}

// probeOllama uses POST /api/generate (Ollama native) for minimal latency.
func (p *DefaultLatencyProber) probeOllama(ctx context.Context, baseURL, modelID string) error {
	body := map[string]any{
		"model":   modelID,
		"prompt":  "Hi",
		"stream":  false,
		"options": map[string]any{"num_predict": 1},
	}
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/generate", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

// probeOpenAICompatible uses POST /v1/chat/completions with minimal messages.
func (p *DefaultLatencyProber) probeOpenAICompatible(ctx context.Context, baseURL, modelID string) error {
	body := map[string]any{
		"model":   modelID,
		"messages": []map[string]any{{"role": "user", "content": "Hi"}},
		"max_tokens": 1,
		"stream": false,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chat completions %d: %s", resp.StatusCode, string(b))
	}
	return nil
}
