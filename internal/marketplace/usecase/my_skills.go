package usecase

import (
	"context"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/entity"
	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/port"
)

// MySkills lists published skills and earnings (publisher dashboard).
// Phase 2: add earnings aggregation.
type MySkills struct {
	Registry port.RegistryPort
}

// Execute returns the publisher's skills. Phase 1 uses stub registry.
func (u *MySkills) Execute(ctx context.Context) ([]entity.Skill, error) {
	return u.Registry.ListMySkills()
}
