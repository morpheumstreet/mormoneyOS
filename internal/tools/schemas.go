package tools

import (
	"github.com/morpheumlabs/mormoneyos-go/internal/inference"
)

// BuiltinToolSchemas returns OpenAI-format tool definitions for the default built-in tools.
// Used as fallback when a non-Registry ToolExecutor is passed to the agent loop.
func BuiltinToolSchemas() []inference.ToolDefinition {
	return NewRegistry().Schemas()
}
