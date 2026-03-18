package usecase

import (
	"context"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/port"
)

// InstallSkill triggers safe install with permission manifest check.
// Phase 2: VC badge + MORM reward claim logic.
type InstallSkill struct {
	Registry port.RegistryPort
	// OnChain port for Phase 2
}

// Execute performs permission check and install. Phase 1 returns stub success.
func (u *InstallSkill) Execute(ctx context.Context, skillID string, agentCardSig string) (string, error) {
	// Phase 1: stub — Phase 2 adds permission manifest check, VC badge, MORM reward
	_ = u.Registry
	_ = skillID
	_ = agentCardSig
	return "installed", nil
}
