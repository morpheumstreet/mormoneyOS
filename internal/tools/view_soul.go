package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// ViewSoulTool displays the soul/identity document.
type ViewSoulTool struct {
	Store ToolStore
}

func (ViewSoulTool) Name() string        { return "view_soul" }
func (ViewSoulTool) Description() string { return "View your soul document (identity, values, constraints)." }
func (ViewSoulTool) Parameters() string {
	return `{"type":"object","properties":{},"required":[]}`
}

func (t *ViewSoulTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	// Try KV first (if soul was stored)
	if t.Store != nil {
		if s, ok, _ := t.Store.GetKV("soul_content"); ok && s != "" {
			return s, nil
		}
	}
	// Try SOUL.md in common locations
	paths := []string{"SOUL.md", ".automaton/SOUL.md", "~/.automaton/SOUL.md"}
	for _, p := range paths {
		if strings.HasPrefix(p, "~/") {
			if h, err := os.UserHomeDir(); err == nil {
				p = filepath.Join(h, p[2:])
			}
		}
		data, err := os.ReadFile(p)
		if err == nil {
			return string(data), nil
		}
	}
	return "No soul document found. Create SOUL.md or use update_soul.", nil
}
