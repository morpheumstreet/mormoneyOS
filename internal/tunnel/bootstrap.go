package tunnel

import (
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// NewFromConfig builds ProviderRegistry and TunnelManager from config.
// If cfg is nil or has no enabled providers, returns registry with bore and custom (when configured).
func NewFromConfig(cfg *types.TunnelConfig) (*ProviderRegistry, *TunnelManager) {
	reg := NewProviderRegistry()
	store := NewActiveTunnelStore()
	mgr := NewTunnelManager(reg, store)

	if cfg == nil {
		// Default: bore only
		reg.Register(BoreProvider())
		return reg, mgr
	}

	// Register enabled providers
	providers := cfg.Providers
	if providers == nil {
		providers = make(map[string]types.TunnelProviderConfig)
	}

	// bore: enabled by default if not explicitly disabled
	if p, ok := providers["bore"]; !ok || p.Enabled {
		reg.Register(BoreProvider())
	}

	// custom: when startCommand is set
	if p, ok := providers["custom"]; ok && p.Enabled && p.StartCommand != "" {
		reg.Register(CustomProvider(p.StartCommand, p.URLPattern, ""))
	}

	return reg, mgr
}
