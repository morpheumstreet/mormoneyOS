// Package marketplace provides mormaegis tool adapters (thin MCP layer).
package marketplace

import (
	"context"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/dto"
	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/usecase"
	"github.com/morpheumlabs/mormoneyos-go/internal/tools"
)

// SearchTool implements mormaegis.search.
type SearchTool struct {
	UseCase *usecase.SearchSkills
}

var _ tools.Tool = (*SearchTool)(nil)

func (t *SearchTool) Name() string { return "mormaegis.search" }
func (t *SearchTool) Description() string {
	return "Search for Perp Trading Skills, Mirofish Decision Packs, or Prediction Resolution Skills"
}
func (t *SearchTool) Parameters() string {
	return `{"type":"object","properties":{"query":{"type":"string","description":"Search query"},"filter":{"type":"string","description":"Optional filter (e.g. perp_ready)"}},"required":["query"]}`
}

func (t *SearchTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	query, _ := args["query"].(string)
	filter := ""
	if f, ok := args["filter"].(string); ok {
		filter = f
	}
	skills, err := t.UseCase.Execute(ctx, query, filter)
	if err != nil {
		return "", err
	}
	return dto.FormatSkills(skills), nil
}
