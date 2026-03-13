package inference

import (
	"context"
)

// StubClient returns empty responses (no real inference).
// Used when no API keys are configured or for testing.
type StubClient struct {
	Model string
}

// NewStubClient creates a stub inference client.
func NewStubClient(model string) *StubClient {
	if model == "" {
		model = "stub"
	}
	return &StubClient{Model: model}
}

// Chat returns an empty response with no tool calls.
func (s *StubClient) Chat(ctx context.Context, messages []ChatMessage, opts *InferenceOptions) (*InferenceResponse, error) {
	_ = ctx
	_ = opts
	return &InferenceResponse{
		Content:      "[stub] No inference client configured. Use OpenAI or Conway API keys for real inference.",
		ToolCalls:    nil,
		InputTokens:  0,
		OutputTokens: 0,
		FinishReason: "stop",
		CostCents:    0,
	}, nil
}

// GetDefaultModel returns the configured model name.
func (s *StubClient) GetDefaultModel() string {
	return s.Model
}

// SetLowComputeMode is a no-op for stub.
func (s *StubClient) SetLowComputeMode(enabled bool) {
	_ = enabled
}

// Ensure StubClient implements Client.
var _ Client = (*StubClient)(nil)
