package tunnel

import (
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// PopulateRegistry registers providers from config into the given registry.
// Used by NewFromConfig and TunnelManager.Reload.
func PopulateRegistry(reg *ProviderRegistry, cfg *types.TunnelConfig) {
	if cfg == nil {
		reg.Register(BoreProvider())
		return
	}
	providers := cfg.Providers
	if providers == nil {
		providers = make(map[string]types.TunnelProviderConfig)
	}
	if p, ok := providers["bore"]; !ok || p.Enabled {
		reg.Register(BoreProvider())
	}
	if p, ok := providers["localtunnel"]; !ok || p.Enabled {
		reg.Register(LocaltunnelProvider())
	}
	if p, ok := providers["cloudflare"]; ok && p.Enabled && p.Token != "" {
		if prov := CloudflareProvider(p); prov != nil {
			reg.Register(prov)
		}
	}
	if p, ok := providers["ngrok"]; ok && p.Enabled && p.AuthToken != "" {
		if prov := NgrokProvider(p); prov != nil {
			reg.Register(prov)
		}
	}
	if p, ok := providers["tailscale"]; ok && p.Enabled && p.AuthKey != "" {
		if prov := TailscaleProvider(p); prov != nil {
			reg.Register(prov)
		}
	}
	if p, ok := providers["custom"]; ok && p.Enabled && p.StartCommand != "" {
		reg.Register(CustomProvider(p.StartCommand, p.URLPattern, ""))
	}
}

// NewFromConfig builds ProviderRegistry and TunnelManager from config.
// If cfg is nil or has no enabled providers, returns registry with bore and custom (when configured).
func NewFromConfig(cfg *types.TunnelConfig) (*ProviderRegistry, *TunnelManager) {
	reg := NewProviderRegistry()
	store := NewActiveTunnelStore()
	mgr := NewTunnelManager(reg, store)
	PopulateRegistry(reg, cfg)
	return reg, mgr
}
