// Package usecase holds business logic only (pure Go, no HTTP).
package usecase

import (
	"context"
	"fmt"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/entity"
	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/port"
)

// NegotiateOffer posts or counter-offers on a skill.
// Phase 2: on-chain offer flow via OnChainPort.
type NegotiateOffer struct {
	Registry port.RegistryPort
	OnChain  port.OnChainPort
}

// Execute creates or updates an offer. When offerID is empty, posts new offer (requires skillID); else counter-offers.
func (u *NegotiateOffer) Execute(ctx context.Context, skillID string, offerID string, mormAmount float64) (*entity.Offer, error) {
	if mormAmount < 0 {
		return nil, fmt.Errorf("morm_amount must be >= 0")
	}
	if u.OnChain == nil {
		return &entity.Offer{
			ID:         offerID,
			SkillID:    skillID,
			MORMAmount: mormAmount,
			Status:     "pending",
		}, nil
	}
	var offer *entity.Offer
	var err error
	if offerID == "" {
		if skillID == "" {
			return nil, fmt.Errorf("skill_id required for new offer")
		}
		offer, err = u.OnChain.PostOffer(skillID, mormAmount)
	} else {
		offer, err = u.OnChain.CounterOffer(offerID, mormAmount)
	}
	if err != nil {
		return nil, err
	}
	return offer, nil
}
