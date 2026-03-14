package skills

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/morpheumlabs/mormoneyos-go/internal/config"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// SkillInserter inserts a skill into the store (minimal interface for installer).
type SkillInserter interface {
	InsertSkill(name, description, instructions, source, path string, enabled bool) error
}

// RegistryConfigFrom resolves registry URL and timeout from config.
func RegistryConfigFrom(cfg *types.SkillsConfig) (url string, timeoutSec int) {
	url = "https://clawhub.ai"
	timeoutSec = 30
	if cfg != nil && cfg.Registry != nil {
		if cfg.Registry.URL != "" {
			url = cfg.Registry.URL
		}
		if cfg.Registry.TimeoutSec > 0 {
			timeoutSec = cfg.Registry.TimeoutSec
		}
	}
	return url, timeoutSec
}

// TrustedRootsFrom returns trusted skill roots from config.
func TrustedRootsFrom(cfg *types.SkillsConfig) []string {
	defaultRoot := config.ResolvePath("~/.automaton/skills")
	if cfg == nil || len(cfg.TrustedRoots) == 0 {
		return []string{defaultRoot}
	}
	return cfg.TrustedRoots
}

// InstallFromRegistry fetches a skill from ClawHub and installs it. Single place for registry install logic.
// Returns (skillRoot, skillName, error).
func InstallFromRegistry(ctx context.Context, client *RegistryClient, store SkillInserter, cfg *types.SkillsConfig, slug, version, name, desc string) (skillRoot, skillName string, err error) {
	if slug == "" {
		return "", "", fmt.Errorf("slug required")
	}
	meta, resolvedVersion, err := client.Resolve(ctx, slug)
	if err != nil {
		return "", "", fmt.Errorf("resolve skill: %w", err)
	}
	if version != "" {
		resolvedVersion = version
	}
	zipData, err := client.Download(ctx, slug, resolvedVersion)
	if err != nil {
		return "", "", fmt.Errorf("download skill: %w", err)
	}
	roots := TrustedRootsFrom(cfg)
	targetRoot := roots[0]
	if targetRoot == "" {
		targetRoot = config.ResolvePath("~/.automaton/skills")
	}
	if err := os.MkdirAll(targetRoot, 0755); err != nil {
		return "", "", fmt.Errorf("create skills dir: %w", err)
	}
	targetDir := filepath.Join(targetRoot, slug)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", "", fmt.Errorf("create skill dir: %w", err)
	}
	skillRoot, err = ExtractZipToDir(zipData, targetDir)
	if err != nil {
		_ = os.RemoveAll(targetDir)
		return "", "", fmt.Errorf("extract skill: %w", err)
	}
	skillName = name
	if skillName == "" {
		skillName = meta.DisplayName
	}
	if skillName == "" {
		skillName = slug
	}
	skillDesc := desc
	if skillDesc == "" {
		skillDesc = meta.Summary
	}
	if err := store.InsertSkill(skillName, skillDesc, "", "registry", skillRoot, true); err != nil {
		return "", "", fmt.Errorf("install skill: %w", err)
	}
	return skillRoot, skillName, nil
}
