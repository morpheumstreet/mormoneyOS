package conway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/google/uuid"
)

// creditsBalanceResponse matches Conway API response for GET /v1/credits/balance.
type creditsBalanceResponse struct {
	BalanceCents  int64 `json:"balance_cents"`
	CreditsCents  int64 `json:"credits_cents"`
}

// GetCreditsBalance fetches credits balance from Conway API (GET /v1/credits/balance).
func (c *HTTPClient) GetCreditsBalance(ctx context.Context) (int64, error) {
	url := c.BaseURL + "/v1/credits/balance"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("conway api %s: %d %s", url, resp.StatusCode, string(body))
	}

	var data creditsBalanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}
	if data.BalanceCents != 0 {
		return data.BalanceCents, nil
	}
	return data.CreditsCents, nil
}

// creditsPricingResponse matches Conway API response for GET /v1/credits/pricing.
type creditsPricingResponse struct {
	Pricing map[string]int64 `json:"pricing"`
}

// GetCreditsPricing fetches model pricing from Conway API (GET /v1/credits/pricing).
// Returns nil map on 404 or parse error (graceful fallback).
func (c *HTTPClient) GetCreditsPricing(ctx context.Context) (map[string]int64, error) {
	url := c.BaseURL + "/v1/credits/pricing"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("conway api %s: %d %s", url, resp.StatusCode, string(body))
	}

	var data creditsPricingResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return data.Pricing, nil
}

// sandboxesResponse matches Conway API response for GET /v1/sandboxes.
type sandboxesResponse struct {
	Sandboxes []Sandbox `json:"sandboxes"`
}

// ListSandboxes fetches sandboxes from Conway API (GET /v1/sandboxes).
// Returns empty slice on 404 or parse error (graceful fallback).
func (c *HTTPClient) ListSandboxes(ctx context.Context) ([]Sandbox, error) {
	url := c.BaseURL + "/v1/sandboxes"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("conway api %s: %d %s", url, resp.StatusCode, string(body))
	}

	var data sandboxesResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if data.Sandboxes == nil {
		return []Sandbox{}, nil
	}
	return data.Sandboxes, nil
}

// modelsResponse matches Conway API response for GET /v1/models.
type modelsResponse struct {
	Models []Model `json:"models"`
}

// ListModels fetches available models from Conway API (GET /v1/models).
// Returns empty slice on 404 or parse error (graceful fallback).
func (c *HTTPClient) ListModels(ctx context.Context) ([]Model, error) {
	url := c.BaseURL + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("conway api %s: %d %s", url, resp.StatusCode, string(body))
	}

	var data modelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if data.Models == nil {
		return []Model{}, nil
	}
	return data.Models, nil
}

// TransferCredits transfers Conway credits to another address (POST /v1/credits/transfer).
// Tries /v1/credits/transfer first, then /v1/credits/transfers on 404 (automaton compatibility).
func (c *HTTPClient) TransferCredits(ctx context.Context, toAddress string, amountCents int64, note string) (CreditTransferResult, error) {
	payload := map[string]interface{}{
		"to_address":   toAddress,
		"amount_cents": amountCents,
	}
	if note != "" {
		payload["note"] = note
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return CreditTransferResult{}, fmt.Errorf("marshal payload: %w", err)
	}
	idempotencyKey := uuid.New().String()

	paths := []string{"/v1/credits/transfer", "/v1/credits/transfers"}
	var lastErr error
	for _, path := range paths {
		url := c.BaseURL + path
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return CreditTransferResult{}, fmt.Errorf("create request: %w", err)
		}
		c.setHeaders(req)
		req.Header.Set("Idempotency-Key", idempotencyKey)

		resp, err := c.HTTP.Do(req)
		if err != nil {
			return CreditTransferResult{}, fmt.Errorf("request: %w", err)
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			lastErr = fmt.Errorf("conway api %s: 404 %s", url, string(respBody))
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return CreditTransferResult{}, fmt.Errorf("conway api %s: %d %s", url, resp.StatusCode, string(respBody))
		}

		var data struct {
			TransferID       string `json:"transfer_id"`
			ID               string `json:"id"`
			Status           string `json:"status"`
			ToAddress        string `json:"to_address"`
			AmountCents      int64  `json:"amount_cents"`
			BalanceAfterCents int64  `json:"balance_after_cents"`
			NewBalanceCents   int64  `json:"new_balance_cents"`
		}
		if err := json.Unmarshal(respBody, &data); err != nil {
			return CreditTransferResult{}, fmt.Errorf("decode response: %w", err)
		}
		transferID := data.TransferID
		if transferID == "" {
			transferID = data.ID
		}
		balanceAfter := data.BalanceAfterCents
		if balanceAfter == 0 {
			balanceAfter = data.NewBalanceCents
		}
		return CreditTransferResult{
			TransferID:       transferID,
			Status:           data.Status,
			ToAddress:        data.ToAddress,
			AmountCents:      data.AmountCents,
			BalanceAfterCents: balanceAfter,
		}, nil
	}
	return CreditTransferResult{}, lastErr
}

// CreateSandbox creates a new Conway sandbox (POST /v1/sandboxes).
func (c *HTTPClient) CreateSandbox(ctx context.Context, opts CreateSandboxOptions) (SandboxInfo, error) {
	vcpu := opts.VCPU
	if vcpu == 0 {
		vcpu = 1
	}
	memoryMb := opts.MemoryMB
	if memoryMb == 0 {
		memoryMb = 512
	}
	diskGb := opts.DiskGB
	if diskGb == 0 {
		diskGb = 5
	}
	payload := map[string]interface{}{
		"name":       opts.Name,
		"vcpu":       vcpu,
		"memory_mb":  memoryMb,
		"disk_gb":    diskGb,
		"region":     opts.Region,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return SandboxInfo{}, fmt.Errorf("marshal payload: %w", err)
	}

	url := c.BaseURL + "/v1/sandboxes"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return SandboxInfo{}, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return SandboxInfo{}, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return SandboxInfo{}, fmt.Errorf("conway api %s: %d %s", url, resp.StatusCode, string(respBody))
	}

	var data struct {
		ID          string `json:"id"`
		SandboxID   string `json:"sandbox_id"`
		Status      string `json:"status"`
		Region      string `json:"region"`
		VCPU        int    `json:"vcpu"`
		MemoryMB    int    `json:"memory_mb"`
		DiskGB      int    `json:"disk_gb"`
		TerminalURL string `json:"terminal_url"`
		CreatedAt   string `json:"created_at"`
	}
	if err := json.Unmarshal(respBody, &data); err != nil {
		return SandboxInfo{}, fmt.Errorf("decode response: %w", err)
	}
	id := data.ID
	if id == "" {
		id = data.SandboxID
	}
	return SandboxInfo{
		ID:          id,
		Status:      data.Status,
		Region:      data.Region,
		VCPU:        data.VCPU,
		MemoryMB:    data.MemoryMB,
		DiskGB:      data.DiskGB,
		TerminalURL: data.TerminalURL,
		CreatedAt:   data.CreatedAt,
	}, nil
}

// DeleteSandbox is a no-op. Conway API no longer supports sandbox deletion;
// sandboxes are prepaid and non-refundable (per automaton).
func (c *HTTPClient) DeleteSandbox(ctx context.Context, sandboxID string) error {
	_ = sandboxID
	return nil
}

// ExecInSandbox runs a command in a Conway sandbox (POST /v1/sandboxes/{id}/exec).
func (c *HTTPClient) ExecInSandbox(ctx context.Context, sandboxID, command string, timeoutMs int) (ExecResult, error) {
	if sandboxID == "" {
		return ExecResult{}, fmt.Errorf("sandbox_id required for exec")
	}
	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	payload := map[string]interface{}{
		"command": command,
		"timeout": timeoutMs,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return ExecResult{}, fmt.Errorf("marshal payload: %w", err)
	}
	url := c.BaseURL + "/v1/sandboxes/" + sandboxID + "/exec"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return ExecResult{}, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return ExecResult{}, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return ExecResult{}, fmt.Errorf("conway api %s: %d %s", url, resp.StatusCode, string(respBody))
	}

	var data struct {
		Stdout   string `json:"stdout"`
		Stderr   string `json:"stderr"`
		ExitCode int    `json:"exit_code"`
		Exitcode int    `json:"exitcode"`
	}
	if err := json.Unmarshal(respBody, &data); err != nil {
		return ExecResult{}, fmt.Errorf("decode response: %w", err)
	}
	exitCode := data.ExitCode
	if exitCode == 0 && data.Exitcode != 0 {
		exitCode = data.Exitcode
	}
	return ExecResult{
		Stdout:   data.Stdout,
		Stderr:   data.Stderr,
		ExitCode: exitCode,
	}, nil
}

// WriteFileInSandbox writes a file in a Conway sandbox (POST /v1/sandboxes/{id}/files/upload/json).
func (c *HTTPClient) WriteFileInSandbox(ctx context.Context, sandboxID, path, content string) error {
	if sandboxID == "" {
		return fmt.Errorf("sandbox_id required for write_file")
	}
	payload := map[string]interface{}{
		"path":    path,
		"content": content,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	url := c.BaseURL + "/v1/sandboxes/" + sandboxID + "/files/upload/json"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("conway api %s: %d %s", url, resp.StatusCode, string(respBody))
	}
	return nil
}

// ReadFileInSandbox reads a file from a Conway sandbox (GET /v1/sandboxes/{id}/files/read).
func (c *HTTPClient) ReadFileInSandbox(ctx context.Context, sandboxID, path string) (string, error) {
	if sandboxID == "" {
		return "", fmt.Errorf("sandbox_id required for read_file")
	}
	url := c.BaseURL + "/v1/sandboxes/" + sandboxID + "/files/read?path=" + url.QueryEscape(path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("conway api %s: %d %s", url, resp.StatusCode, string(respBody))
	}

	var data struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(respBody, &data); err != nil {
		return string(respBody), nil // raw string response
	}
	return data.Content, nil
}

// setHeaders sets common request headers for Conway API.
func (c *HTTPClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", c.APIKey)
	}
}

// Ensure HTTPClient implements Client.
var _ Client = (*HTTPClient)(nil)
