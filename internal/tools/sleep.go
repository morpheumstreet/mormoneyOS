package tools

import (
	"context"
	"fmt"
	"time"
)

// SleepTool enters sleep mode for a specified duration.
type SleepTool struct {
	Store ToolStore
}

func (SleepTool) Name() string { return "sleep" }
func (SleepTool) Description() string {
	return "Enter sleep mode for a specified duration. Heartbeat continues running."
}
func (SleepTool) Parameters() string {
	return `{"type":"object","properties":{"duration_seconds":{"type":"number","description":"How long to sleep in seconds"},"reason":{"type":"string","description":"Why you are sleeping"}},"required":["duration_seconds"]}`
}

func (t *SleepTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "sleep requires store"}
	}
	dur, ok := args["duration_seconds"].(float64)
	if !ok {
		return "", ErrInvalidArgs{Msg: "duration_seconds required"}
	}
	reason, _ := args["reason"].(string)
	if reason == "" {
		reason = "No reason given"
	}
	until := time.Now().Add(time.Duration(dur) * time.Second)
	_ = t.Store.SetAgentState("sleeping")
	_ = t.Store.SetKV("sleep_until", until.UTC().Format(time.RFC3339))
	_ = t.Store.SetKV("sleep_reason", reason)
	return fmt.Sprintf("Entering sleep mode for %v seconds. Reason: %s. Heartbeat will continue.", int(dur), reason), nil
}
