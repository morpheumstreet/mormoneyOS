package tools

import (
	"context"
	"fmt"
	"strings"
)

// ListSkillsTool lists installed skills.
type ListSkillsTool struct {
	Store ToolStore
}

func (ListSkillsTool) Name() string        { return "list_skills" }
func (ListSkillsTool) Description() string { return "List installed skills with name, description, and enabled status." }
func (ListSkillsTool) Parameters() string {
	return `{"type":"object","properties":{},"required":[]}`
}

func (t *ListSkillsTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "list_skills requires store"}
	}
	skills, ok := t.Store.GetSkills()
	if !ok || len(skills) == 0 {
		return "No skills installed.", nil
	}
	var sb strings.Builder
	for i, s := range skills {
		if i > 0 {
			sb.WriteString("\n")
		}
		name, _ := s["name"].(string)
		desc, _ := s["description"].(string)
		enabled := true
		switch e := s["enabled"].(type) {
		case bool:
			enabled = e
		case int64:
			enabled = e != 0
		case int:
			enabled = e != 0
		}
		status := "disabled"
		if enabled {
			status = "enabled"
		}
		sb.WriteString(fmt.Sprintf("- %s: %s [%s]", name, desc, status))
	}
	return sb.String(), nil
}
