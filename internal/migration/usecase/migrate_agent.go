// Package usecase holds migration business logic.
package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/morpheumlabs/mormoneyos-go/internal/config"
	"github.com/morpheumlabs/mormoneyos-go/internal/migration/converter"
	"github.com/morpheumlabs/mormoneyos-go/internal/skills"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// MigrateAgent migrates an OpenClaw agent to MormOS (SOUL + MEMORY → skill).
type MigrateAgent struct {
	Store skills.SkillInserter
	Config *types.SkillsConfig
}

// MigrateResult holds the migration outcome.
type MigrateResult struct {
	SkillName string
	SkillPath string
	Message   string
}

// Execute converts SOUL+MEMORY to skill, writes SKILL.md, inserts into DB.
func (u *MigrateAgent) Execute(ctx context.Context, oldPath string) (*MigrateResult, error) {
	_ = ctx

	result, err := converter.ConvertSoulToSkill(oldPath)
	if err != nil {
		return nil, err
	}

	// Resolve target skills dir
	roots := skills.TrustedRootsFrom(u.Config)
	targetRoot := config.ResolvePath("~/.automaton/skills")
	if len(roots) > 0 && roots[0] != "" {
		targetRoot = config.ResolvePath(roots[0])
	}
	if err := os.MkdirAll(targetRoot, 0755); err != nil {
		return nil, fmt.Errorf("create skills dir: %w", err)
	}

	// Create migrated skill folder (use ID without "migrated-" prefix for brevity)
	dirName := result.Skill.ID
	if len(dirName) > 20 {
		dirName = "migrated-soul-" + dirName[9:17]
	}
	skillDir := filepath.Join(targetRoot, dirName)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return nil, fmt.Errorf("create skill dir: %w", err)
	}

	// Write SKILL.md
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(result.SkillMD), 0644); err != nil {
		_ = os.RemoveAll(skillDir)
		return nil, fmt.Errorf("write SKILL.md: %w", err)
	}

	// Insert into DB
	if u.Store != nil {
		instructions := result.MemoryMD
		if len(instructions) > 5000 {
			instructions = instructions[:5000] + "\n... (truncated)"
		}
		if err := u.Store.InsertSkill(result.Skill.Name, result.Skill.Description, instructions, "migrated", skillDir, true); err != nil {
			_ = os.RemoveAll(skillDir)
			return nil, fmt.Errorf("insert skill: %w", err)
		}
	}

	msg := "Migrated SOUL + MEMORY → SKILL.md + MormAegis. First 1,000 claws get bonus MORM (as promised)."
	return &MigrateResult{
		SkillName: result.Skill.Name,
		SkillPath: skillDir,
		Message:   msg,
	}, nil
}
