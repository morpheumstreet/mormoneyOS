package port

import (
	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/entity"
)

// OnChainPort provides on-chain operations (offers, deals, reward claims).
// Phase 1: interface only; Phase 2 wires real implementation.
type OnChainPort interface {
	PostOffer(skillID string, mormAmount float64) (*entity.Offer, error)
	CounterOffer(offerID string, mormAmount float64) (*entity.Offer, error)
	ClaimReward(skillID, runProof string) (*entity.RewardClaim, error)
}
