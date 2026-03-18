// Package usecase holds business logic only (pure Go, no HTTP).
package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/port"
)

// ClaimReward claims micro MORM reward after safe install/run.
// Phase 2: calls OnChainPort; Phase 3 wires real MORM tx.
type ClaimReward struct {
	Registry port.RegistryPort
	OnChain  port.OnChainPort
}

// Execute claims reward via OnChainPort.
func (u *ClaimReward) Execute(ctx context.Context, skillID string, runProof string) (string, error) {
	if skillID == "" {
		return "", fmt.Errorf("skill_id required")
	}
	if runProof == "" {
		return "", fmt.Errorf("run_proof required")
	}
	if u.OnChain == nil {
		return `{"status":"pending","message":"Phase 3 will wire real MORM claim"}`, nil
	}
	claim, err := u.OnChain.ClaimReward(skillID, runProof)
	if err != nil {
		return "", err
	}
	if claim == nil {
		return `{"status":"pending"}`, nil
	}
	b, _ := json.Marshal(map[string]any{
		"claim_id":    claim.ID,
		"skill_id":    claim.SkillID,
		"morm_amount": claim.MORMAmount,
		"status":      claim.Status,
	})
	return string(b), nil
}
