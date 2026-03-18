package usecase

import (
	"context"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/entity"
	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/port"
)

// NegotiateOffer posts or counter-offers on a skill.
// Phase 2: on-chain offer flow via OnChainPort.
type NegotiateOffer struct {
	Registry port.RegistryPort
	OnChain  port.OnChainPort
}

// Execute creates or updates an offer. Phase 1 returns stub.
func (u *NegotiateOffer) Execute(ctx context.Context, offerID string, mormAmount float64) (*entity.Offer, error) {
	_ = u.OnChain
	_ = offerID
	_ = mormAmount
	// Phase 1: stub — Phase 2 wires OnChain.PostOffer / OnChain.CounterOffer
	return &entity.Offer{
		ID:         offerID,
		MORMAmount: mormAmount,
		Status:     "pending",
	}, nil
}
