package inference

import (
	"context"
	"sync"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// InferenceClientHolder holds the current inference client and supports atomic reload.
// Thread-safe; safe for concurrent reads and reloads.
type InferenceClientHolder struct {
	mu     sync.RWMutex
	client Client
}

// NewInferenceClientHolder creates a holder with the client from cfg.
func NewInferenceClientHolder(cfg *types.AutomatonConfig) *InferenceClientHolder {
	h := &InferenceClientHolder{}
	h.Reload(cfg)
	return h
}

// Client returns the current inference client. Safe for concurrent use.
func (h *InferenceClientHolder) Client() Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.client
}

// Reload creates a new client from cfg and swaps it atomically.
func (h *InferenceClientHolder) Reload(cfg *types.AutomatonConfig) {
	newClient := NewClientFromConfig(cfg)
	h.mu.Lock()
	defer h.mu.Unlock()
	h.client = newClient
}

// LiveClient returns an inference.Client that always delegates to the holder's current client.
// Use this when injecting into the agent loop and web server.
func (h *InferenceClientHolder) LiveClient() Client {
	return &liveInferenceClient{holder: h}
}

type liveInferenceClient struct {
	holder *InferenceClientHolder
}

func (c *liveInferenceClient) Chat(ctx context.Context, messages []ChatMessage, opts *InferenceOptions) (*InferenceResponse, error) {
	return c.holder.Client().Chat(ctx, messages, opts)
}

func (c *liveInferenceClient) GetDefaultModel() string {
	return c.holder.Client().GetDefaultModel()
}

func (c *liveInferenceClient) SetLowComputeMode(enabled bool) {
	c.holder.Client().SetLowComputeMode(enabled)
}
