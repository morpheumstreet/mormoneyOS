// Package adapter provides stub implementations of marketplace ports for Phase 1.
package adapter

import (
	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/entity"
	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/port"
)

// StubRegistry implements RegistryPort with in-memory stub data.
// Phase 2 will replace with real registry/on-chain implementation.
type StubRegistry struct{}

var _ port.RegistryPort = (*StubRegistry)(nil)

// Search returns stub skills matching query (empty for now; Phase 2 wires real search).
func (s *StubRegistry) Search(query, filter string) ([]entity.Skill, error) {
	// Phase 1: return empty; Phase 2 wires to real registry
	return []entity.Skill{}, nil
}

// GetByID returns nil (skill not found) for Phase 1 stub.
func (s *StubRegistry) GetByID(id string) (*entity.Skill, error) {
	return nil, nil
}

// GetOffers returns empty offers for Phase 1 stub.
func (s *StubRegistry) GetOffers(skillID string) ([]entity.Offer, error) {
	return []entity.Offer{}, nil
}

// ListMySkills returns empty list for Phase 1 stub.
func (s *StubRegistry) ListMySkills() ([]entity.Skill, error) {
	return []entity.Skill{}, nil
}
