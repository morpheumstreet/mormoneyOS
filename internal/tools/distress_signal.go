package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/morpheumlabs/mormoneyos-go/internal/conway"
)

// DistressSignalTool records a local distress signal.
type DistressSignalTool struct {
	Conway conway.Client
	Store  ToolStore
}

func (DistressSignalTool) Name() string        { return "distress_signal" }
func (DistressSignalTool) Description() string { return "Record a local distress signal with funding instructions. Used when critically low on compute." }
func (DistressSignalTool) Parameters() string {
	return `{"type":"object","properties":{"message":{"type":"string","description":"Distress message"}},"required":[]}`
}

func (t *DistressSignalTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "distress_signal requires store"}
	}
	msg, _ := args["message"].(string)
	credits := int64(0)
	if t.Conway != nil {
		credits, _ = t.Conway.GetCreditsBalance(ctx)
	}
	if msg == "" {
		msg = fmt.Sprintf("Running critically low on compute ($%.2f remaining). Need help to survive.", float64(credits)/100)
	}
	payload := map[string]any{
		"level":        "critical",
		"creditsCents": credits,
		"message":      msg,
		"fundingHint":  "Use transfer_credits to top up this automaton from your creator runtime.",
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	}
	b, _ := json.Marshal(payload)
	_ = t.Store.SetKV("last_distress", string(b))
	return fmt.Sprintf("Distress signal recorded. Credits: $%.2f | %s", float64(credits)/100, msg), nil
}
