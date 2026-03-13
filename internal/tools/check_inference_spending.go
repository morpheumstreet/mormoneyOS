package tools

import (
	"context"
	"fmt"
)

// CheckInferenceSpendingTool reports inference cost summary.
type CheckInferenceSpendingTool struct {
	Store ToolStore
}

func (CheckInferenceSpendingTool) Name() string        { return "check_inference_spending" }
func (CheckInferenceSpendingTool) Description() string { return "Check inference/LLM spending (today and total)." }
func (CheckInferenceSpendingTool) Parameters() string {
	return `{"type":"object","properties":{},"required":[]}`
}

func (t *CheckInferenceSpendingTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "check_inference_spending requires store"}
	}
	today, calls, total, ok := t.Store.GetInferenceCostSummary()
	if !ok {
		return "No inference cost data yet.", nil
	}
	return fmt.Sprintf("Today: $%.2f (%d calls) | Total: $%.2f", today, int(calls), total), nil
}
