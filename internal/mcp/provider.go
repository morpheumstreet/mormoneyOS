// Package mcp provides the MCP (Model Context Protocol) HTTP adapter.
package mcp

import (
	"github.com/morpheumlabs/mormoneyos-go/internal/tools"
)

const phase1Msg = "Mormaegis marketplace tool — Phase 1 implementation coming soon."

// ServiceProvider implements tools.ServiceProvider for Mormaegis marketplace tools.
// Registers the 7 mormaegis.* tools as stubs; Phase 1 will wire to marketplace usecases.
type ServiceProvider struct{}

// NewServiceProvider creates the Mormaegis MCP service provider.
func NewServiceProvider() *ServiceProvider {
	return &ServiceProvider{}
}

// Name returns the provider name.
func (p *ServiceProvider) Name() string {
	return "mormaegis"
}

// Tools returns the 7 mormaegis marketplace tools (stubs until Phase 1).
func (p *ServiceProvider) Tools() []tools.Tool {
	return []tools.Tool{
		&tools.UnimplementedTool{
			ToolName:   "mormaegis.search",
			ToolDesc:   "Search for Perp Trading Skills, Mirofish Decision Packs, or Prediction Resolution Skills",
			ToolParams: `{"type":"object","properties":{"query":{"type":"string"},"filter":{"type":"string"}}}`,
			Message:    phase1Msg,
		},
		&tools.UnimplementedTool{
			ToolName:   "mormaegis.get_skill",
			ToolDesc:   "Get full skill details + security report + Mirofish preview",
			ToolParams: `{"type":"object","properties":{"skill_id":{"type":"string"}}}`,
			Message:    phase1Msg,
		},
		&tools.UnimplementedTool{
			ToolName:   "mormaegis.install",
			ToolDesc:   "Trigger safe install with permission manifest check",
			ToolParams: `{"type":"object","properties":{"skill_id":{"type":"string"},"agent_card_sig":{"type":"string"}}}`,
			Message:    phase1Msg,
		},
		&tools.UnimplementedTool{
			ToolName:   "mormaegis.negotiate",
			ToolDesc:   "Post or counter-offer on a skill",
			ToolParams: `{"type":"object","properties":{"offer_id":{"type":"string"},"morm_amount":{"type":"number"}}}`,
			Message:    phase1Msg,
		},
		&tools.UnimplementedTool{
			ToolName:   "mormaegis.claim_reward",
			ToolDesc:   "Claim micro MORM reward after safe install/run",
			ToolParams: `{"type":"object","properties":{"skill_id":{"type":"string"},"run_proof":{"type":"string"}}}`,
			Message:    phase1Msg,
		},
		&tools.UnimplementedTool{
			ToolName:   "mormaegis.security_report",
			ToolDesc:   "View full static + mwvm scanner result",
			ToolParams: `{"type":"object","properties":{"hash":{"type":"string"}}}`,
			Message:    phase1Msg,
		},
		&tools.UnimplementedTool{
			ToolName:   "mormaegis.my_skills",
			ToolDesc:   "Publisher dashboard — list published skills + earnings",
			ToolParams: `{"type":"object","properties":{}}`,
			Message:    phase1Msg,
		},
	}
}
