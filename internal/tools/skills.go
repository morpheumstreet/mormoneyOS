package tools

import (
	"context"
	"fmt"
	"strings"
)

// SkillStore provides skill DB operations.
type SkillStore interface {
	InsertSkill(name, description, instructions, source, path string, enabled bool) error
	DeleteSkill(name string) error
	GetSkills() ([]map[string]any, bool)
}

// InstallSkillTool installs a skill from a path.
type InstallSkillTool struct {
	Store interface {
		InsertSkill(name, description, instructions, source, path string, enabled bool) error
	}
}

func (InstallSkillTool) Name() string        { return "install_skill" }
func (InstallSkillTool) Description() string { return "Install a skill from a path." }
func (InstallSkillTool) Parameters() string {
	return `{"type":"object","properties":{"name":{"type":"string"},"path":{"type":"string"},"description":{"type":"string"}},"required":["name"]}`
}

func (t *InstallSkillTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "install_skill requires store"}
	}
	name, _ := args["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrInvalidArgs{Msg: "name required"}
	}
	path, _ := args["path"].(string)
	desc, _ := args["description"].(string)
	if err := t.Store.InsertSkill(name, desc, "", "installed", path, true); err != nil {
		return "", fmt.Errorf("install skill: %w", err)
	}
	return fmt.Sprintf("Installed skill %q from %q", name, path), nil
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
