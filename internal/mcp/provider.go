// Package mcp provides the MCP (Model Context Protocol) HTTP adapter.
package mcp

import (
	mkt "github.com/morpheumlabs/mormoneyos-go/internal/marketplace"
	"github.com/morpheumlabs/mormoneyos-go/internal/mcp/tools/marketplace"
	migrateuc "github.com/morpheumlabs/mormoneyos-go/internal/migration/usecase"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
	"github.com/morpheumlabs/mormoneyos-go/internal/tools"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// ServiceProviderOptions configures Phase 2 real adapters.
type ServiceProviderOptions struct {
	SkillsConfig *types.SkillsConfig
	DB           *state.Database // For ListMySkills + Install; nil = stubs
}

// ServiceProvider implements tools.ServiceProvider for Mormaegis marketplace tools.
// Phase 2: uses real RegistryAdapter, ScannerAdapter, OnChainAdapter when options provided.
type ServiceProvider struct {
	tools []tools.Tool
}

// NewServiceProvider creates the Mormaegis MCP service provider.
// Phase 1: stub adapters when no options. Phase 2: real adapters when opts provided.
func NewServiceProvider() *ServiceProvider {
	return NewServiceProviderWithOptions(nil)
}

// NewServiceProviderWithOptions creates provider with real adapters when opts.DB and opts.SkillsConfig set.
func NewServiceProviderWithOptions(opts *ServiceProviderOptions) *ServiceProvider {
	svc := mkt.NewService(
		func() *types.SkillsConfig {
			if opts != nil {
				return opts.SkillsConfig
			}
			return nil
		}(),
		func() *state.Database {
			if opts != nil {
				return opts.DB
			}
			return nil
		}(),
	)

	toolList := []tools.Tool{
		&marketplace.SearchTool{UseCase: svc.SearchSkills},
		&marketplace.GetSkillTool{UseCase: svc.GetSkill},
		&marketplace.InstallTool{UseCase: svc.InstallSkill},
		&marketplace.NegotiateTool{UseCase: svc.NegotiateOffer},
		&marketplace.ClaimRewardTool{UseCase: svc.ClaimReward},
		&marketplace.SecurityReportTool{UseCase: svc.SecurityReport},
		&marketplace.MySkillsTool{UseCase: svc.MySkills},
	}

	// Optional: mormaegis.migrate when DB + config available
	if opts != nil && opts.DB != nil {
		migrateUC := &migrateuc.MigrateAgent{Store: opts.DB, Config: opts.SkillsConfig}
		if migrateUC.Config == nil {
			migrateUC.Config = &types.SkillsConfig{}
		}
		toolList = append(toolList, &marketplace.MigrateTool{UseCase: migrateUC})
	}

	return &ServiceProvider{tools: toolList}
}

// Name returns the provider name.
func (p *ServiceProvider) Name() string {
	return "mormaegis"
}

// Tools returns the 7 mormaegis marketplace tools.
func (p *ServiceProvider) Tools() []tools.Tool {
	return p.tools
}
