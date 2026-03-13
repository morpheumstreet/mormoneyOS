package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/conway"
)

// ListModelsTool lists available inference models (Conway or fallback to built-in).
type ListModelsTool struct {
	Conway conway.Client
}

func (ListModelsTool) Name() string        { return "list_models" }
func (ListModelsTool) Description() string { return "List available inference models and their providers." }
func (ListModelsTool) Parameters() string {
	return `{"type":"object","properties":{},"required":[]}`
}

func (t *ListModelsTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Conway != nil {
		models, err := t.Conway.ListModels(ctx)
		if err == nil && len(models) > 0 {
			var sb strings.Builder
			for i, m := range models {
				if i > 0 {
					sb.WriteString("\n")
				}
				sb.WriteString(fmt.Sprintf("- %s (%s)", m.ID, m.Provider))
			}
			return sb.String(), nil
		}
	}
	return "No models available. Configure Conway or OpenAI for model discovery.", nil
}
