package dto

// ExecuteResponse is the response for POST /mcp (MCP content format).
type ExecuteResponse struct {
	Content []ContentItem `json:"content"`
}

// ContentItem is a single content block (MCP standard).
type ContentItem struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}
