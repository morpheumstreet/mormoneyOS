// Package port defines interfaces for Dependency Inversion.
package port

import "github.com/morpheumlabs/mormoneyos-go/internal/marketplace/entity"

// RegistryPort provides skill search and retrieval.
type RegistryPort interface {
	Search(query, filter string) ([]entity.Skill, error)
	GetByID(id string) (*entity.Skill, error)
	GetOffers(skillID string) ([]entity.Offer, error)
	ListMySkills() ([]entity.Skill, error)
}
