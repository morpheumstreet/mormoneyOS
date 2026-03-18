// Package converter converts OpenClaw SOUL.md + MEMORY.md to MormOS skill format.
package converter

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/entity"
)

const (
	soulFile   = "SOUL.md"
	memoryFile = "MEMORY.md"
)

// ConvertResult holds the converted skill and SKILL.md content.
type ConvertResult struct {
	Skill    entity.Skill
	SkillMD  string // SKILL.md content for writing to disk
	MemoryMD string // MEMORY content (included in instructions)
}

// ConvertSoulToSkill reads SOUL.md and MEMORY.md from oldPath and produces a MormOS skill.
// Returns error if oldPath is invalid or SOUL.md is missing.
func ConvertSoulToSkill(oldPath string) (*ConvertResult, error) {
	oldPath = filepath.Clean(oldPath)
	if oldPath == "" || oldPath == "." {
		return nil, fmt.Errorf("old path required")
	}
	info, err := os.Stat(oldPath)
	if err != nil {
		return nil, fmt.Errorf("path %s: %w", oldPath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path must be a directory: %s", oldPath)
	}

	soulPath := filepath.Join(oldPath, soulFile)
	soulData, err := os.ReadFile(soulPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("SOUL.md not found in %s (OpenClaw agent format required)", oldPath)
		}
		return nil, fmt.Errorf("read SOUL.md: %w", err)
	}

	memoryData, _ := os.ReadFile(filepath.Join(oldPath, memoryFile))
	memoryStr := strings.TrimSpace(string(memoryData))

	// Build SKILL.md v2 format: SOUL content + optional MEMORY section
	var skillMD strings.Builder
	skillMD.WriteString("# Migrated Soul\n\n")
	skillMD.WriteString("> One-click migration from OpenClaw. Original SOUL + MEMORY preserved.\n\n")
	skillMD.WriteString("## Soul\n\n")
	skillMD.WriteString(string(soulData))
	if memoryStr != "" {
		skillMD.WriteString("\n\n## Memory\n\n")
		skillMD.WriteString(memoryStr)
	}

	id := "migrated-" + generateID()
	name := strings.TrimSpace(extractNameFromSoul(string(soulData)))
	if name == "" {
		name = "Imported Soul"
	}

	skill := entity.Skill{
		ID:          id,
		Name:        name,
		Description: string(soulData),
		PriceMORM:   0,
		Badges:      []string{"Migrated", "KYA-Verified"},
		Permissions: map[string]any{"root": false, "wallet": true},
		SecurityHash: "",
		PerpReady:   false,
	}

	return &ConvertResult{
		Skill:    skill,
		SkillMD:  skillMD.String(),
		MemoryMD: memoryStr,
	}, nil
}

func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func extractNameFromSoul(soul string) string {
	// Try to extract name from first # or ## heading
	lines := strings.Split(soul, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
		if strings.HasPrefix(line, "## ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "## "))
		}
	}
	return ""
}
