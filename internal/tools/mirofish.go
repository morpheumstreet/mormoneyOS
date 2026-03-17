package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// MiroFishTool triggers MiroFish swarm intelligence functions (simulate markets, run predictions, get reports, chat with digital crowd).
// Register when MiroFish is enabled in config.
type MiroFishTool struct {
	client *http.Client
	cfg    *types.MiroFishConfig
}

// NewMiroFishTool creates a MiroFish tool from config.
func NewMiroFishTool(cfg *types.MiroFishConfig) *MiroFishTool {
	if cfg == nil {
		cfg = &types.MiroFishConfig{
			Enabled:        true,
			BaseURL:        "http://localhost:5001",
			TimeoutSeconds: 300,
			DefaultLLM:    "qwen-plus",
			MaxAgents:      2000,
		}
	}
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 300 * time.Second
	}
	return &MiroFishTool{
		client: &http.Client{Timeout: timeout},
		cfg:    cfg,
	}
}

func (t *MiroFishTool) Name() string        { return "mirofish" }
func (t *MiroFishTool) Description() string { return "Trigger MiroFish swarm intelligence: run simulations, get ReportAgent output, inject variables, chat with digital crowd. Use for market predictions, foresight, and rehearsal before betting." }
func (t *MiroFishTool) Parameters() string {
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

func (t *MiroFishTool) Execute(ctx context.Context, args map[string]any) (string, error) {
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

	// Map action to MiroFish API path (common conventions)
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

	url := strings.TrimSuffix(t.cfg.BaseURL, "/") + "/api/" + path
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("MiroFish request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("MiroFish %s: %s (status %d)", action, string(respBody), resp.StatusCode)
	}

	// Return clean JSON for agent memory
	if len(respBody) == 0 {
		return `{"status":"ok","action":"` + action + `"}`, nil
	}
	return string(respBody), nil
}
