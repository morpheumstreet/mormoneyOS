// Package dto holds MCP request/response models (DRY — reused with REST layer).
package dto

// ExecuteRequest is the body for POST /mcp (tool execution).
type ExecuteRequest struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}
