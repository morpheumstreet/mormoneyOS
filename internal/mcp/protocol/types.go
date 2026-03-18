// Package protocol defines MCP (Model Context Protocol) spec types.
// Aligned with Anthropic MCP standard for agent-native tool discovery and execution.
package protocol

// Tool is the MCP tool schema for discovery (tools/list).
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema is JSON Schema for tool parameters.
type InputSchema struct {
	Type       string            `json:"type,omitempty"`
	Properties map[string]Schema `json:"properties,omitempty"`
	Required   []string          `json:"required,omitempty"`
}

// Schema is a JSON Schema property.
type Schema struct {
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

// ToolsListResponse is the response for GET /mcp/tools (or tools/list).
type ToolsListResponse struct {
	Tools []Tool `json:"tools"`
}
