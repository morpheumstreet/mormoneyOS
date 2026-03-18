package usecase

import (
	"context"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/port"
)

// ClaimReward claims micro MORM reward after safe install/run.
// Phase 2: on-chain claim + run proof verification.
type ClaimReward struct {
	Registry port.RegistryPort
}

// Execute claims reward. Phase 1 returns stub message.
func (u *ClaimReward) Execute(ctx context.Context, skillID string, runProof string) (string, error) {
	_ = u.Registry
	_ = skillID
	_ = runProof
	return "Phase 2: MORM reward claim will verify run_proof on-chain.", nil
}
