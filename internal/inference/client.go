package inference

import (
	"context"
)

// ChatMessage is a single message in the conversation (TS-aligned).
type ChatMessage struct {
	Role    string `json:"role"` // system, user, assistant
	Content string `json:"content,omitempty"`
	// ToolCalls for assistant messages (OpenAI format)
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents a tool invocation from the model (OpenAI format).
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // "function"
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// InferenceOptions for chat (TS-aligned).
type InferenceOptions struct {
	Model       string                 `json:"model,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Tools       []ToolDefinition       `json:"tools,omitempty"`
	ToolChoice  string                 `json:"tool_choice,omitempty"` // "auto", "none"
	Temperature float64                `json:"temperature,omitempty"`
	Extra       map[string]interface{} `json:"-"`
}

// ToolDefinition is the OpenAI tool schema (TS-aligned).
type ToolDefinition struct {
	Type     string      `json:"type"` // "function"
	Function ToolSchema  `json:"function"`
}

// ToolSchema defines a tool's name and parameters.
type ToolSchema struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  string `json:"parameters,omitempty"` // JSON schema
}

// InferenceResponse is the chat response (TS-aligned).
type InferenceResponse struct {
	Content       string     `json:"content"`
	ToolCalls     []ToolCall `json:"tool_calls,omitempty"`
	InputTokens   int        `json:"input_tokens"`
	OutputTokens  int        `json:"output_tokens"`
	FinishReason  string     `json:"finish_reason"` // "stop", "tool_calls", etc.
	CostCents     int        `json:"cost_cents"`
}

// Client is the inference client interface (TS InferenceClient).
type Client interface {
	Chat(ctx context.Context, messages []ChatMessage, opts *InferenceOptions) (*InferenceResponse, error)
	GetDefaultModel() string
	SetLowComputeMode(enabled bool)
}
