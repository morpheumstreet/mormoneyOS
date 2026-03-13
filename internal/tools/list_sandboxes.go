package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/conway"
)

// ListSandboxesTool lists Conway sandboxes.
// Requires Conway client; register only when Conway is configured.
type ListSandboxesTool struct {
	Conway conway.Client
}

func (t *ListSandboxesTool) Name() string { return "list_sandboxes" }
func (t *ListSandboxesTool) Description() string {
	return "List your Conway sandboxes (VMs). Returns sandbox IDs, names, and status."
}
func (t *ListSandboxesTool) Parameters() string { return `{"type":"object","properties":{},"required":[]}` }

func (t *ListSandboxesTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Conway == nil {
		return "", ErrConwayNotConfigured
	}
	sandboxes, err := t.Conway.ListSandboxes(ctx)
	if err != nil {
		return "", fmt.Errorf("list sandboxes: %w", err)
	}
	if len(sandboxes) == 0 {
		return "No sandboxes found.", nil
	}
	var sb strings.Builder
	for i, s := range sandboxes {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("- %s (id: %s)", s.Name, s.ID))
	}
	return sb.String(), nil
}
