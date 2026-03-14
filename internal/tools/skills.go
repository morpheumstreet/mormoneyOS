package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/config"
	"github.com/morpheumlabs/mormoneyos-go/internal/skills"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

const (
	skillToml = "SKILL.toml"
	skillMd   = "SKILL.md"
)

// SkillStore provides skill DB operations.
type SkillStore interface {
	InsertSkill(name, description, instructions, source, path string, enabled bool) error
	DeleteSkill(name string) error
	GetSkills() ([]map[string]any, bool)
}

// InstallSkillTool installs a skill from a path (file-based) or from ClawHub registry.
// Use create_skill for DB-only (builtin) skills.
type InstallSkillTool struct {
	Store  interface {
		InsertSkill(name, description, instructions, source, path string, enabled bool) error
	}
	Config *types.AutomatonConfig // Optional; for trusted roots validation
}

func (InstallSkillTool) Name() string        { return "install_skill" }
func (InstallSkillTool) Description() string { return "Install a skill from a directory path or from ClawHub registry. Path: local dir with SKILL.md/SKILL.toml. ClawHub: use source=clawhub and id=<slug> (e.g. gmail-secretary). Use create_skill for DB-only skills." }
func (InstallSkillTool) Parameters() string {
	return `{"type":"object","properties":{"name":{"type":"string"},"path":{"type":"string"},"description":{"type":"string"},"source":{"type":"string","enum":["path","clawhub"]},"id":{"type":"string"},"version":{"type":"string"}},"required":[]}`
}

func (t *InstallSkillTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "install_skill requires store"}
	}
	source, _ := args["source"].(string)
	source = strings.TrimSpace(strings.ToLower(source))
	id, _ := args["id"].(string)
	id = strings.TrimSpace(id)
	version, _ := args["version"].(string)
	version = strings.TrimSpace(version)
	path, _ := args["path"].(string)
	path = strings.TrimSpace(path)
	name, _ := args["name"].(string)
	name = strings.TrimSpace(name)
	desc, _ := args["description"].(string)
	desc = strings.TrimSpace(desc)

	// ClawHub install: source=clawhub, id=slug
	if source == "clawhub" || (source == "" && id != "" && path == "") {
		return t.installFromClawHub(ctx, id, version, name, desc)
	}

	// Path install (default)
	if path == "" {
		return "", ErrInvalidArgs{Msg: "path required for install_skill; use source=clawhub and id=<slug> for registry, or create_skill for DB-only"}
	}
	if name == "" {
		return "", ErrInvalidArgs{Msg: "name required for path install"}
	}
	return t.installFromPath(path, name, desc)
}

func (t *InstallSkillTool) installFromClawHub(ctx context.Context, slug, version, name, desc string) (string, error) {
	if slug == "" {
		return "", ErrInvalidArgs{Msg: "id (ClawHub slug) required for registry install"}
	}
	cfg := t.Config
	if cfg == nil || cfg.Skills == nil {
		cfg = &types.AutomatonConfig{Skills: &types.SkillsConfig{}}
	}
	regURL, timeoutSec := skills.RegistryConfigFrom(cfg.Skills)
	client := skills.NewRegistryClient(regURL, timeoutSec)
	skillRoot, skillName, err := skills.InstallFromRegistry(ctx, client, t.Store, cfg.Skills, slug, version, name, desc)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Installed skill %q from ClawHub (%s) to %q", skillName, slug, skillRoot), nil
}

func (t *InstallSkillTool) installFromPath(path, name, desc string) (string, error) {
	// Normalize to directory: store directory only
	dir := path
	if strings.HasSuffix(filepath.Clean(path), skillMd) || strings.HasSuffix(filepath.Clean(path), skillToml) {
		dir = filepath.Dir(path)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("path resolve failed: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrInvalidArgs{Msg: "skill directory does not exist"}
		}
		return "", fmt.Errorf("path stat: %w", err)
	}
	if !info.IsDir() {
		return "", ErrInvalidArgs{Msg: "path must be a directory"}
	}
	// Must contain SKILL.md or SKILL.toml
	hasToml := false
	hasMd := false
	if _, err := os.Stat(filepath.Join(resolved, skillToml)); err == nil {
		hasToml = true
	}
	if _, err := os.Stat(filepath.Join(resolved, skillMd)); err == nil {
		hasMd = true
	}
	if !hasToml && !hasMd {
		return "", ErrInvalidArgs{Msg: "skill directory must contain SKILL.md or SKILL.toml"}
	}
	// Trusted roots check
	trusted := []string{config.ResolvePath("~/.automaton/skills")}
	if t.Config != nil && t.Config.Skills != nil && len(t.Config.Skills.TrustedRoots) > 0 {
		trusted = t.Config.Skills.TrustedRoots
	}
	allowed := false
	for _, root := range trusted {
		r := filepath.Clean(root)
		if r == "" {
			continue
		}
		if strings.HasPrefix(r, "~") {
			home, _ := os.UserHomeDir()
			r = home + strings.TrimPrefix(r, "~")
		}
		absRoot, _ := filepath.Abs(r)
		if absRoot != "" && (resolved == absRoot || strings.HasPrefix(resolved, absRoot+string(filepath.Separator))) {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", ErrInvalidArgs{Msg: "skill directory must be under a trusted root (e.g. ~/.automaton/skills)"}
	}

	if err := t.Store.InsertSkill(name, desc, "", "installed", resolved, true); err != nil {
		return "", fmt.Errorf("install skill: %w", err)
	}
	return fmt.Sprintf("Installed skill %q from %q", name, resolved), nil
}

// CreateSkillTool creates a new skill with description.
type CreateSkillTool struct {
	Store interface {
		InsertSkill(name, description, instructions, source, path string, enabled bool) error
	}
}

func (CreateSkillTool) Name() string        { return "create_skill" }
func (CreateSkillTool) Description() string { return "Create a new skill with name and description." }
func (CreateSkillTool) Parameters() string {
	return `{"type":"object","properties":{"name":{"type":"string"},"description":{"type":"string"},"instructions":{"type":"string"}},"required":["name"]}`
}

func (t *CreateSkillTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "create_skill requires store"}
	}
	name, _ := args["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrInvalidArgs{Msg: "name required"}
	}
	desc, _ := args["description"].(string)
	instructions, _ := args["instructions"].(string)
	if err := t.Store.InsertSkill(name, desc, instructions, "builtin", "", true); err != nil {
		return "", fmt.Errorf("create skill: %w", err)
	}
	return fmt.Sprintf("Created skill %q", name), nil
}

// RemoveSkillTool removes a skill by name.
type RemoveSkillTool struct {
	Store interface {
		DeleteSkill(name string) error
	}
}

func (RemoveSkillTool) Name() string        { return "remove_skill" }
func (RemoveSkillTool) Description() string { return "Remove a skill by name." }
func (RemoveSkillTool) Parameters() string {
	return `{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`
}

func (t *RemoveSkillTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "remove_skill requires store"}
	}
	name, _ := args["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrInvalidArgs{Msg: "name required"}
	}
	if err := t.Store.DeleteSkill(name); err != nil {
		return "", fmt.Errorf("remove skill: %w", err)
	}
	return fmt.Sprintf("Removed skill %q", name), nil
}
