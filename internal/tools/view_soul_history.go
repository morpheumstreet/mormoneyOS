package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const soulHistoryKey = "soul_history"
const maxHistoryEntries = 10

type soulHistoryEntry struct {
	At      string `json:"at"`
	Preview string `json:"preview"`
}

// ViewSoulHistoryTool shows recent soul edits.
type ViewSoulHistoryTool struct {
	Store ToolStore
}

func (ViewSoulHistoryTool) Name() string        { return "view_soul_history" }
func (ViewSoulHistoryTool) Description() string { return "View history of soul document edits." }
func (ViewSoulHistoryTool) Parameters() string {
	return `{"type":"object","properties":{},"required":[]}`
}

func (t *ViewSoulHistoryTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "No store configured.", nil
	}
	raw, ok, _ := t.Store.GetKV(soulHistoryKey)
	if !ok || raw == "" {
		return "No soul history.", nil
	}
	var entries []soulHistoryEntry
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return "Corrupt history.", nil
	}
	var out []string
	for i, e := range entries {
		out = append(out, fmt.Sprintf("%d. %s: %s", i+1, e.At, e.Preview))
	}
	return strings.Join(out, "\n"), nil
}
