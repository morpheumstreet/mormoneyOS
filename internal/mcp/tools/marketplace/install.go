package marketplace

import (
	"context"
	"fmt"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/usecase"
	"github.com/morpheumlabs/mormoneyos-go/internal/tools"
)

// InstallTool implements mormaegis.install.
type InstallTool struct {
	UseCase *usecase.InstallSkill
}

var _ tools.Tool = (*InstallTool)(nil)

func (t *InstallTool) Name() string { return "mormaegis.install" }
func (t *InstallTool) Description() string {
	return "Trigger safe install with permission manifest check"
}
func (t *InstallTool) Parameters() string {
	return `{"type":"object","properties":{"skill_id":{"type":"string"},"agent_card_sig":{"type":"string"}},"required":["skill_id"]}`
}

func (t *InstallTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	skillID, _ := args["skill_id"].(string)
	if skillID == "" {
		return "", fmt.Errorf("skill_id is required")
	}
	agentCardSig, _ := args["agent_card_sig"].(string)
	result, err := t.UseCase.Execute(ctx, skillID, agentCardSig)
	if err != nil {
		return "", err
	}
	return result, nil
}
