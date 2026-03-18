package marketplace

import (
	"context"
	"fmt"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/dto"
	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/usecase"
	"github.com/morpheumlabs/mormoneyos-go/internal/tools"
)

// GetSkillTool implements mormaegis.get_skill.
type GetSkillTool struct {
	UseCase *usecase.GetSkill
}

var _ tools.Tool = (*GetSkillTool)(nil)

func (t *GetSkillTool) Name() string { return "mormaegis.get_skill" }
func (t *GetSkillTool) Description() string {
	return "Get full skill details + security report + Mirofish preview"
}
func (t *GetSkillTool) Parameters() string {
	return `{"type":"object","properties":{"skill_id":{"type":"string","description":"Skill ID"}},"required":["skill_id"]}`
}

func (t *GetSkillTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	skillID, _ := args["skill_id"].(string)
	if skillID == "" {
		return "", fmt.Errorf("skill_id is required")
	}
	skill, err := t.UseCase.Execute(ctx, skillID)
	if err != nil {
		return "", err
	}
	return dto.FormatSkill(skill), nil
}
