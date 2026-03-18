// Package mcp provides the MCP (Model Context Protocol) HTTP adapter.
package mcp

import (
	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/adapter"
	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/usecase"
	"github.com/morpheumlabs/mormoneyos-go/internal/mcp/tools/marketplace"
	"github.com/morpheumlabs/mormoneyos-go/internal/tools"
)

// ServiceProvider implements tools.ServiceProvider for Mormaegis marketplace tools.
// Registers the 7 mormaegis.* tools wired to marketplace usecases (Phase 1: stub ports).
type ServiceProvider struct {
	reg    *adapter.StubRegistry
	scan   *adapter.StubScanner
	tools  []tools.Tool
}

// NewServiceProvider creates the Mormaegis MCP service provider.
// Phase 1 uses stub registry/scanner; Phase 2 will accept real port implementations.
func NewServiceProvider() *ServiceProvider {
	reg := &adapter.StubRegistry{}
	scan := &adapter.StubScanner{}

	searchUC := &usecase.SearchSkills{Registry: reg, Scanner: scan}
	getSkillUC := &usecase.GetSkill{Registry: reg, Scanner: scan}
	installUC := &usecase.InstallSkill{Registry: reg}
	negotiateUC := &usecase.NegotiateOffer{Registry: reg, OnChain: nil}
	claimUC := &usecase.ClaimReward{Registry: reg}
	securityUC := &usecase.SecurityReport{Scanner: scan}
	mySkillsUC := &usecase.MySkills{Registry: reg}

	toolList := []tools.Tool{
		&marketplace.SearchTool{UseCase: searchUC},
		&marketplace.GetSkillTool{UseCase: getSkillUC},
		&marketplace.InstallTool{UseCase: installUC},
		&marketplace.NegotiateTool{UseCase: negotiateUC},
		&marketplace.ClaimRewardTool{UseCase: claimUC},
		&marketplace.SecurityReportTool{UseCase: securityUC},
		&marketplace.MySkillsTool{UseCase: mySkillsUC},
	}

	return &ServiceProvider{
		reg:   reg,
		scan:  scan,
		tools: toolList,
	}
}

// Name returns the provider name.
func (p *ServiceProvider) Name() string {
	return "mormaegis"
}

// Tools returns the 7 mormaegis marketplace tools.
func (p *ServiceProvider) Tools() []tools.Tool {
	return p.tools
}
