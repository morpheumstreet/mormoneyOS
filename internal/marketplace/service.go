// Package marketplace provides the Mormaegis marketplace domain.
// Service wires adapters and usecases for use by MCP and REST layers.
package marketplace

import (
	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/adapter"
	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/port"
	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/usecase"
	"github.com/morpheumlabs/mormoneyos-go/internal/skills"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// Service holds marketplace usecases wired to real adapters.
// Shared by MCP tools and REST API (DRY).
type Service struct {
	SearchSkills    *usecase.SearchSkills
	GetSkill        *usecase.GetSkill
	InstallSkill    *usecase.InstallSkill
	NegotiateOffer  *usecase.NegotiateOffer
	ClaimReward     *usecase.ClaimReward
	SecurityReport  *usecase.SecurityReport
	MySkills        *usecase.MySkills
}

// NewService creates a marketplace service with real adapters.
// When cfg or db is nil, uses stubs.
func NewService(cfg *types.SkillsConfig, db *state.Database) *Service {
	var reg port.RegistryPort
	var scan port.ScannerPort
	var onchain port.OnChainPort
	var installer port.InstallerPort

	if cfg != nil && db != nil {
		regURL, timeout := skills.RegistryConfigFrom(cfg)
		client := skills.NewRegistryClient(regURL, timeout)
		reg = adapter.NewRegistryAdapter(client, db, cfg)
		scan = adapter.NewScannerAdapter()
		onchain = adapter.NewOnChainAdapter()
		installer = adapter.NewInstallerAdapter(client, db, cfg)
	} else {
		reg = &adapter.StubRegistry{}
		scan = &adapter.StubScanner{}
		onchain = nil
		installer = nil
	}

	return &Service{
		SearchSkills:  &usecase.SearchSkills{Registry: reg, Scanner: scan},
		GetSkill:     &usecase.GetSkill{Registry: reg, Scanner: scan},
		InstallSkill: &usecase.InstallSkill{Registry: reg, Scanner: scan, Installer: installer, OnChain: onchain},
		NegotiateOffer: &usecase.NegotiateOffer{Registry: reg, OnChain: onchain},
		ClaimReward:   &usecase.ClaimReward{Registry: reg, OnChain: onchain},
		SecurityReport: &usecase.SecurityReport{Scanner: scan},
		MySkills:      &usecase.MySkills{Registry: reg},
	}
}
