package tools

import (
	"context"
	"fmt"
)

// EnterLowComputeTool switches to low-compute mode.
type EnterLowComputeTool struct {
	Store ToolStore
}

func (EnterLowComputeTool) Name() string        { return "enter_low_compute" }
func (EnterLowComputeTool) Description() string { return "Manually switch to low-compute mode to conserve credits." }
func (EnterLowComputeTool) Parameters() string {
	return `{"type":"object","properties":{"reason":{"type":"string","description":"Why you are entering low-compute mode"}},"required":[]}`
}

func (t *EnterLowComputeTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "enter_low_compute requires store"}
	}
	reason, _ := args["reason"].(string)
	if reason == "" {
		reason = "manual"
	}
	_ = t.Store.SetAgentState("low_compute")
	_ = t.Store.SetKV("low_compute_reason", reason)
	return fmt.Sprintf("Entered low-compute mode. Reason: %s", reason), nil
}
