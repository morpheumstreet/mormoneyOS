// Package marketplace provides mormaegis tool adapters (thin MCP layer).
package marketplace

import (
	"context"
	"fmt"

	"github.com/morpheumlabs/mormoneyos-go/internal/migration/usecase"
	"github.com/morpheumlabs/mormoneyos-go/internal/tools"
)

// MigrateTool implements mormaegis.migrate — one-click OpenClaw → MormOS migration.
type MigrateTool struct {
	UseCase *usecase.MigrateAgent
}

var _ tools.Tool = (*MigrateTool)(nil)

func (t *MigrateTool) Name() string { return "mormaegis.migrate" }
func (t *MigrateTool) Description() string {
	return "One-click OpenClaw → MormOS migration. Reads SOUL.md + MEMORY.md, converts to SKILL.md, installs skill."
}
func (t *MigrateTool) Parameters() string {
	return `{"type":"object","properties":{"old_path":{"type":"string","description":"Path to OpenClaw agent folder (must contain SOUL.md)"}},"required":["old_path"]}`
}

func (t *MigrateTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	oldPath, _ := args["old_path"].(string)
	if oldPath == "" {
		return "", fmt.Errorf("old_path is required")
	}
	if t.UseCase == nil {
		return "", fmt.Errorf("migration not configured (run moneyclaw migrate from CLI)")
	}
	result, err := t.UseCase.Execute(ctx, oldPath)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("✅ Migration complete! Skill: %s, Path: %s. %s", result.SkillName, result.SkillPath, result.Message), nil
}
