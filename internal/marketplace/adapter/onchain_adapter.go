// Package adapter provides real implementations of marketplace ports for Phase 2.
package adapter

import (
	"fmt"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/entity"
	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/port"
)

// OnChainAdapter implements OnChainPort. Phase 2: stub with mock responses.
// Phase 3 will wire real MORM contract + Conway/x402 for rewards.
type OnChainAdapter struct{}

var _ port.OnChainPort = (*OnChainAdapter)(nil)

// NewOnChainAdapter creates an on-chain adapter.
func NewOnChainAdapter() *OnChainAdapter {
	return &OnChainAdapter{}
}

// PostOffer creates a new offer. Phase 2: returns stub.
func (a *OnChainAdapter) PostOffer(skillID string, mormAmount float64) (*entity.Offer, error) {
	return &entity.Offer{
		ID:         "offer-" + skillID,
		SkillID:    skillID,
		MORMAmount: mormAmount,
		Status:     "pending",
	}, nil
}

// CounterOffer updates an existing offer. Phase 2: returns stub.
func (a *OnChainAdapter) CounterOffer(offerID string, mormAmount float64) (*entity.Offer, error) {
	return &entity.Offer{
		ID:         offerID,
		MORMAmount: mormAmount,
		Status:     "pending",
	}, nil
}

// ClaimReward claims MORM reward. Phase 2: returns stub claim; Phase 3 wires real MORM tx.
func (a *OnChainAdapter) ClaimReward(skillID, runProof string) (*entity.RewardClaim, error) {
	if skillID == "" || runProof == "" {
		return nil, fmt.Errorf("skill_id and run_proof required")
	}
	return &entity.RewardClaim{
		ID:         "claim-" + skillID,
		SkillID:    skillID,
		RunProof:   runProof,
		MORMAmount: 0.05,
		Status:     "claimed",
	}, nil
}
