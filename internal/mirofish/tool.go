package mirofish

import (
	"context"
	"fmt"

	"github.com/morpheumlabs/mormoneyos-go/internal/tools"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// Tool triggers MiroFish swarm intelligence (simulate markets, run predictions, get reports, chat with digital crowd).
type Tool struct {
	client Client
	cfg    *types.MiroFishConfig
}

// NewTool creates a MiroFish tool from config.
func NewTool(cfg *types.MiroFishConfig) *Tool {
	if cfg == nil {
		cfg = &types.MiroFishConfig{
			Enabled:        true,
			BaseURL:        "http://localhost:5001",
			TimeoutSeconds: 300,
			DefaultLLM:    "qwen-plus",
			MaxAgents:      2000,
		}
	}
	return &Tool{
		client: NewHTTPClient(cfg),
		cfg:    cfg,
	}
}

// Ensure Tool implements tools.Tool.
var _ tools.Tool = (*Tool)(nil)

func (t *Tool) Name() string        { return "mirofish" }
func (t *Tool) Description() string { return "Trigger MiroFish swarm intelligence: run simulations, get ReportAgent output, inject variables, chat with digital crowd. Use for market predictions, foresight, and rehearsal before betting." }
func (t *Tool) Parameters() string {
	return `{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["run_simulation", "get_report", "inject_variable", "chat_with_agent", "test_connection"],
				"description": "Action to perform."
			},
			"question": {
				"type": "string",
				"description": "Question for simulation or report (e.g. market prediction)."
			},
			"seeds": {
				"type": "array",
				"items": {"type": "string"},
				"description": "Seed inputs for context (e.g. news, sentiment)."
			},
			"payload": {
				"type": "object",
				"description": "Optional full payload for custom requests."
			}
		},
		"required": ["action"]
	}`
}

func (t *Tool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.cfg == nil || !t.cfg.Enabled {
		return "", fmt.Errorf("MiroFish not configured or disabled")
	}
	action, ok := args["action"].(string)
	if !ok || action == "" {
		return "", fmt.Errorf("action is required")
	}

	// Build payload from args or use provided payload
	payload := make(map[string]any)
	if p, ok := args["payload"].(map[string]any); ok && len(p) > 0 {
		payload = p
	} else {
		payload["question"] = args["question"]
		if seeds, ok := args["seeds"].([]any); ok {
			strs := make([]string, 0, len(seeds))
			for _, s := range seeds {
				if s, ok := s.(string); ok {
					strs = append(strs, s)
				}
			}
			payload["seeds"] = strs
		}
		payload["llm"] = t.cfg.DefaultLLM
		payload["max_agents"] = t.cfg.MaxAgents
	}

	// Map action to MiroFish API path
	path := action
	switch action {
	case "run_simulation":
		path = "simulate"
	case "get_report":
		path = "report"
	case "inject_variable":
		path = "inject"
	case "chat_with_agent":
		path = "chat"
	case "test_connection":
		path = "heartbeat"
	}

	respBody, statusCode, err := t.client.Call(ctx, path, payload)
	if err != nil {
		return "", fmt.Errorf("MiroFish request failed: %w", err)
	}

	if statusCode >= 400 {
		return "", fmt.Errorf("MiroFish %s: %s (status %d)", action, string(respBody), statusCode)
	}

	if len(respBody) == 0 {
		return `{"status":"ok","action":"` + action + `"}`, nil
	}
	return string(respBody), nil
}
