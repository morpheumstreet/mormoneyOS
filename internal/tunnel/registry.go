package tunnel

import "sync"

// ProviderRegistry holds registered tunnel providers. No lifecycle.
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]TunnelProvider
}

// NewProviderRegistry creates an empty provider registry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{providers: make(map[string]TunnelProvider)}
}

// Register adds a provider.
func (r *ProviderRegistry) Register(p TunnelProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
}

// Get returns a provider by name.
func (r *ProviderRegistry) Get(name string) (TunnelProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	return p, ok
}

// List returns all registered provider names.
func (r *ProviderRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.providers))
	for n := range r.providers {
		names = append(names, n)
	}
	return names
}

// Clear removes all providers. Used when reloading config.
func (r *ProviderRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers = make(map[string]TunnelProvider)
}
