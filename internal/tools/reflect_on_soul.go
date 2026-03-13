package tools

import (
	"context"
	"fmt"

	"github.com/morpheumlabs/mormoneyos-go/internal/soul"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

// ReflectOnSoulTool runs the soul reflection pipeline and returns alignment, suggestions, and evidence.
type ReflectOnSoulTool struct {
	Store ToolStore
}

func (ReflectOnSoulTool) Name() string        { return "reflect_on_soul" }
func (ReflectOnSoulTool) Description() string { return "Reflect on your soul document (identity, values, constraints). Computes genesis alignment and suggests updates." }
func (ReflectOnSoulTool) Parameters() string {
	return `{"type":"object","properties":{},"required":[]}`
}

func (t *ReflectOnSoulTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "No store configured; cannot reflect on soul.", nil
	}
	db, ok := t.Store.(*state.Database)
	if !ok {
		s, ok, _ := t.Store.GetKV("soul_content")
		if !ok || s == "" {
			return "No soul document stored. Use update_soul to create one.", nil
		}
		return s, nil
	}
	ref, err := soul.ReflectOnSoul(db)
	if err != nil {
		return fmt.Sprintf("Reflection failed: %v", err), nil
	}
	out := fmt.Sprintf("Soul reflection complete. Alignment: %.2f. Auto-updated: %v", ref.CurrentAlignment, ref.AutoUpdated)
	if len(ref.SuggestedUpdates) > 0 {
		out += fmt.Sprintf(". Suggested updates: %d", len(ref.SuggestedUpdates))
		for _, u := range ref.SuggestedUpdates {
			out += fmt.Sprintf("\n- %s: %s (suggest: %s)", u.Section, u.Reason, truncate(u.SuggestedContent, 80))
		}
	}
	return out, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
