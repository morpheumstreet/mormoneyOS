package usecase

import (
	"context"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/entity"
	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/port"
)

// GetSkill retrieves full skill details plus security badges and optional Mirofish preview.
type GetSkill struct {
	Registry port.RegistryPort
	Scanner  port.ScannerPort
}

// Execute returns the skill with badges; nil if not found.
func (u *GetSkill) Execute(ctx context.Context, skillID string) (*entity.Skill, error) {
	skill, err := u.Registry.GetByID(skillID)
	if err != nil || skill == nil {
		return nil, err
	}
	if u.Scanner != nil && skill.SecurityHash != "" {
		skill.Badges = u.Scanner.GetBadges(skill.SecurityHash)
	}
	return skill, nil
}
