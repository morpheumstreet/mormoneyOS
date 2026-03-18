package marketplace

import (
	"context"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/dto"
	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/usecase"
	"github.com/morpheumlabs/mormoneyos-go/internal/tools"
)

// MySkillsTool implements mormaegis.my_skills.
type MySkillsTool struct {
	UseCase *usecase.MySkills
}

var _ tools.Tool = (*MySkillsTool)(nil)

func (t *MySkillsTool) Name() string { return "mormaegis.my_skills" }
func (t *MySkillsTool) Description() string {
	return "Publisher dashboard — list published skills + earnings"
}
func (t *MySkillsTool) Parameters() string {
	return `{"type":"object","properties":{}}`
}

func (t *MySkillsTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	skills, err := t.UseCase.Execute(ctx)
	if err != nil {
		return "", err
	}
	return dto.FormatSkills(skills), nil
}
