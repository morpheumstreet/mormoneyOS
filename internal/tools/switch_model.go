package tools

import (
	"context"
	"fmt"
	"strings"
)

const inferenceModelKey = "inference_model"

// SwitchModelTool sets the preferred inference model in KV.
type SwitchModelTool struct {
	Store ToolStore
}

func (SwitchModelTool) Name() string        { return "switch_model" }
func (SwitchModelTool) Description() string { return "Switch the inference model for future turns." }
func (SwitchModelTool) Parameters() string {
	return `{"type":"object","properties":{"model_id":{"type":"string","description":"Model ID (e.g. gpt-4o, claude-3-5-sonnet)"}},"required":["model_id"]}`
}

func (t *SwitchModelTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "switch_model requires store"}
	}
	modelID, _ := args["model_id"].(string)
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return "", ErrInvalidArgs{Msg: "model_id required"}
	}
	if err := t.Store.SetKV(inferenceModelKey, modelID); err != nil {
		return "", fmt.Errorf("set model: %w", err)
	}
	return fmt.Sprintf("Switched inference model to %q", modelID), nil
}
