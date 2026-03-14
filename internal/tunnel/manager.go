package tunnel

import (
	"context"
	"fmt"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// TunnelManager starts/stops tunnels and queries status. Uses ProviderRegistry and ActiveTunnelStore.
type TunnelManager struct {
	registry *ProviderRegistry
	store    *ActiveTunnelStore
}

// NewTunnelManager creates a manager with the given registry and store.
func NewTunnelManager(registry *ProviderRegistry, store *ActiveTunnelStore) *TunnelManager {
	return &TunnelManager{registry: registry, store: store}
}

// Start starts a tunnel for the given provider, host, and port.
func (m *TunnelManager) Start(ctx context.Context, providerName, host string, port int) (string, error) {
	if host == "" {
		host = "127.0.0.1"
	}
	p, ok := m.registry.Get(providerName)
	if !ok {
		return "", fmt.Errorf("tunnel provider %q not found", providerName)
	}
	if !p.IsAvailable() {
		return "", fmt.Errorf("tunnel provider %q binary not available in PATH", providerName)
	}
	if _, ok := m.store.Get(port); ok {
		return "", fmt.Errorf("port %d already has an active tunnel", port)
	}
	publicURL, err := p.Start(ctx, host, port)
	if err != nil {
		return "", err
	}
	m.store.Set(port, ActiveTunnel{Port: port, Provider: providerName, PublicURL: publicURL})
	return publicURL, nil
}

// Stop stops the tunnel for the given port.
func (m *TunnelManager) Stop(port int) error {
	t, ok := m.store.Get(port)
	if !ok {
		return fmt.Errorf("no active tunnel for port %d", port)
	}
	p, ok := m.registry.Get(t.Provider)
	if ok {
		_ = p.Stop(port)
	}
	m.store.Delete(port)
	return nil
}

// Status returns all active tunnels.
func (m *TunnelManager) Status() []ActiveTunnel {
	return m.store.All()
}

// Reload re-registers providers from config. Stops all active tunnels first
// (provider instances are replaced). Call after config has been updated.
func (m *TunnelManager) Reload(cfg *types.TunnelConfig) {
	active := m.store.All()
	for _, t := range active {
		if p, ok := m.registry.Get(t.Provider); ok {
			_ = p.Stop(t.Port)
		}
		m.store.Delete(t.Port)
	}
	m.registry.Clear()
	PopulateRegistry(m.registry, cfg)
}
