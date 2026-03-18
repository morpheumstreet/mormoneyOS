package marketplace

import (
	"context"
	"fmt"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/usecase"
	"github.com/morpheumlabs/mormoneyos-go/internal/tools"
)

// ClaimRewardTool implements mormaegis.claim_reward.
type ClaimRewardTool struct {
	UseCase *usecase.ClaimReward
}

var _ tools.Tool = (*ClaimRewardTool)(nil)

func (t *ClaimRewardTool) Name() string { return "mormaegis.claim_reward" }
func (t *ClaimRewardTool) Description() string {
	return "Claim micro MORM reward after safe install/run"
}
func (t *ClaimRewardTool) Parameters() string {
	return `{"type":"object","properties":{"skill_id":{"type":"string"},"run_proof":{"type":"string"}},"required":["skill_id","run_proof"]}`
}

func (t *ClaimRewardTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	skillID, _ := args["skill_id"].(string)
	if skillID == "" {
		return "", fmt.Errorf("skill_id is required")
	}
	runProof, _ := args["run_proof"].(string)
	if runProof == "" {
		return "", fmt.Errorf("run_proof is required")
	}
	return t.UseCase.Execute(ctx, skillID, runProof)
}
