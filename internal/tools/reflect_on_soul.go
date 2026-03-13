package tools

import (
	"context"
)

// ReflectOnSoulTool returns the soul document for reflection (same source as view_soul).
type ReflectOnSoulTool struct {
	Store ToolStore
}

func (ReflectOnSoulTool) Name() string        { return "reflect_on_soul" }
func (ReflectOnSoulTool) Description() string { return "Reflect on your soul document (identity, values, constraints)." }
func (ReflectOnSoulTool) Parameters() string {
	return `{"type":"object","properties":{},"required":[]}`
}

func (t *ReflectOnSoulTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "No store configured; cannot reflect on soul.", nil
	}
	s, ok, _ := t.Store.GetKV("soul_content")
	if !ok || s == "" {
		return "No soul document stored. Use update_soul to create one.", nil
	}
	return s, nil
}
