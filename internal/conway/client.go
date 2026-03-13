package conway

import (
	"context"
	"net/http"
	"time"
)

// ExecResult is the result of ExecInSandbox.
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Client provides Conway API operations per mormoneyOS design.
type Client interface {
	GetCreditsBalance(ctx context.Context) (int64, error)
	GetCreditsPricing(ctx context.Context) (map[string]int64, error)
	ListSandboxes(ctx context.Context) ([]Sandbox, error)
	ListModels(ctx context.Context) ([]Model, error)
	TransferCredits(ctx context.Context, toAddress string, amountCents int64, note string) (CreditTransferResult, error)
	CreateSandbox(ctx context.Context, opts CreateSandboxOptions) (SandboxInfo, error)
	DeleteSandbox(ctx context.Context, sandboxID string) error
	// Sandbox-scoped operations for child runtime (deploy, start, message, verify).
	ExecInSandbox(ctx context.Context, sandboxID, command string, timeoutMs int) (ExecResult, error)
	WriteFileInSandbox(ctx context.Context, sandboxID, path, content string) error
	ReadFileInSandbox(ctx context.Context, sandboxID, path string) (string, error)
}

// CreateSandboxOptions configures sandbox creation (POST /v1/sandboxes).
type CreateSandboxOptions struct {
	Name     string `json:"name"`
	VCPU     int    `json:"vcpu,omitempty"`
	MemoryMB int    `json:"memory_mb,omitempty"`
	DiskGB   int    `json:"disk_gb,omitempty"`
	Region   string `json:"region,omitempty"`
}

// CreditTransferResult is the response from TransferCredits.
type CreditTransferResult struct {
	TransferID       string `json:"transfer_id"`
	Status           string `json:"status"`
	ToAddress        string `json:"to_address"`
	AmountCents       int64  `json:"amount_cents"`
	BalanceAfterCents int64  `json:"balance_after_cents,omitempty"`
}

// SandboxInfo is the response from CreateSandbox.
type SandboxInfo struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Region      string `json:"region"`
	VCPU        int    `json:"vcpu"`
	MemoryMB    int    `json:"memory_mb"`
	DiskGB      int    `json:"disk_gb"`
	TerminalURL string `json:"terminal_url,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}

// Sandbox represents a Conway sandbox.
type Sandbox struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Model represents an available inference model.
type Model struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
}

// HTTPClient is a resilient HTTP client for Conway API.
type HTTPClient struct {
	BaseURL   string
	APIKey    string
	UserAgent string
	HTTP      *http.Client
}

// NewHTTPClient creates a Conway HTTP client.
func NewHTTPClient(baseURL, apiKey string) *HTTPClient {
	return &HTTPClient{
		BaseURL:   baseURL,
		APIKey:    apiKey,
		UserAgent: "moneyclaw-go/0.1.0",
		HTTP: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}
