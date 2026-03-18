// Package mcp provides the MCP (Model Context Protocol) HTTP adapter.
// Mounted at /mcp on the same port 8080 as DashOS. Agent-native, standardized tool-calling surface.
package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/morpheumlabs/mormoneyos-go/internal/inference"
	"github.com/morpheumlabs/mormoneyos-go/internal/mcp/dto"
	"github.com/morpheumlabs/mormoneyos-go/internal/mcp/protocol"
	"github.com/morpheumlabs/mormoneyos-go/internal/tools"
)

// Handler handles MCP HTTP endpoints (GET /mcp/tools, POST /mcp).
type Handler struct {
	ToolsLister ToolsLister
	Executor    Executor
	Log         *slog.Logger
}

// ToolsLister lists available tools (same interface as web.ToolsLister).
type ToolsLister interface {
	List() []string
	Schemas() []inference.ToolDefinition
}

// Executor runs tools by name (same interface as tools.Executor).
type Executor interface {
	Execute(ctx context.Context, name string, args map[string]any) (string, error)
}

// NewHandler creates an MCP handler with the given dependencies.
func NewHandler(lister ToolsLister, exec Executor, log *slog.Logger) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{ToolsLister: lister, Executor: exec, Log: log}
}

// ToolsList handles GET /mcp/tools — returns MCP-format tool list.
func (h *Handler) ToolsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.ToolsLister == nil {
		writeJSON(w, protocol.ToolsListResponse{Tools: nil}, http.StatusOK)
		return
	}
	schemas := h.ToolsLister.Schemas()
	toolList := make([]protocol.Tool, 0, len(schemas))
	for _, s := range schemas {
		if s.Function.Name == "" {
			continue
		}
		t := protocol.Tool{
			Name:        s.Function.Name,
			Description: s.Function.Description,
			InputSchema: protocol.InputSchema{
				Type:       "object",
				Properties: nil,
			},
		}
		if s.Function.Parameters != "" {
			var schema map[string]any
			if err := json.Unmarshal([]byte(s.Function.Parameters), &schema); err == nil {
				if props, ok := schema["properties"].(map[string]any); ok {
					ps := make(map[string]protocol.Schema)
					for k, v := range props {
						if m, ok := v.(map[string]any); ok {
							ps[k] = protocol.Schema{
								Type:        stringVal(m, "type"),
								Description: stringVal(m, "description"),
							}
						}
					}
					t.InputSchema.Properties = ps
				}
			}
		}
		toolList = append(toolList, t)
	}
	writeJSON(w, protocol.ToolsListResponse{Tools: toolList}, http.StatusOK)
}

func stringVal(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Execute handles POST /mcp — executes a tool and returns MCP content format.
func (h *Handler) Execute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Executor == nil {
		writeExecuteError(w, "MCP executor not configured", http.StatusServiceUnavailable)
		return
	}
	var req dto.ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeExecuteError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		writeExecuteError(w, "missing tool name", http.StatusBadRequest)
		return
	}
	if req.Arguments == nil {
		req.Arguments = make(map[string]any)
	}
	ctx := r.Context()
	result, err := h.Executor.Execute(ctx, req.Name, req.Arguments)
	if err != nil {
		if _, ok := err.(tools.ErrUnknownTool); ok {
			writeExecuteError(w, err.Error(), http.StatusNotFound)
			return
		}
		h.Log.Warn("mcp execute failed", "tool", req.Name, "err", err)
		writeExecuteResponse(w, result, err.Error(), http.StatusOK)
		return
	}
	writeExecuteResponse(w, result, "", http.StatusOK)
}

func writeJSON(w http.ResponseWriter, v any, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeExecuteError(w http.ResponseWriter, msg string, status int) {
	writeExecuteResponse(w, "", msg, status)
}

func writeExecuteResponse(w http.ResponseWriter, result, errMsg string, status int) {
	text := result
	if errMsg != "" {
		if text != "" {
			text = text + "\n\nError: " + errMsg
		} else {
			text = "Error: " + errMsg
		}
	}
	writeJSON(w, dto.ExecuteResponse{
		Content: []dto.ContentItem{{Type: "text", Text: text}},
	}, status)
}
