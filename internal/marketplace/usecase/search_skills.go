// Package usecase holds business logic only (pure Go, no HTTP).
package usecase

import (
	"context"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/entity"
	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/port"
)

// SearchSkills searches the registry and optionally enriches with security badges.
type SearchSkills struct {
	Registry port.RegistryPort
	Scanner  port.ScannerPort
}

// Execute runs the search. Pure business logic — no HTTP, no Conway.
func (u *SearchSkills) Execute(ctx context.Context, query string, filter string) ([]entity.Skill, error) {
	skills, err := u.Registry.Search(query, filter)
	if err != nil {
		return nil, err
	}
	for i := range skills {
		if u.Scanner != nil && skills[i].SecurityHash != "" {
			skills[i].Badges = u.Scanner.GetBadges(skills[i].SecurityHash)
		}
	}
	return skills, nil
}
