package mirofish

import (
	"github.com/morpheumlabs/mormoneyos-go/internal/tools"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// ServiceProvider implements tools.ServiceProvider for MiroFish.
// Registers the mirofish tool when MiroFish is enabled in config.
type ServiceProvider struct {
	cfg *types.MiroFishConfig
}

// NewServiceProvider creates a MiroFish service provider.
func NewServiceProvider(cfg *types.MiroFishConfig) *ServiceProvider {
	return &ServiceProvider{cfg: cfg}
}

// Name returns the provider name.
func (p *ServiceProvider) Name() string {
	return "mirofish"
}

// Tools returns MiroFish tools when enabled; otherwise nil.
func (p *ServiceProvider) Tools() []tools.Tool {
	if p.cfg == nil || !p.cfg.Enabled {
		return nil
	}
	return []tools.Tool{NewTool(p.cfg)}
}
