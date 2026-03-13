package tunnel

import (
	"sync"
)

// ActiveTunnelStore holds active tunnels (port → ActiveTunnel).
type ActiveTunnelStore struct {
	mu       sync.RWMutex
	tunnels  map[int]ActiveTunnel
}

// NewActiveTunnelStore creates an empty store.
func NewActiveTunnelStore() *ActiveTunnelStore {
	return &ActiveTunnelStore{tunnels: make(map[int]ActiveTunnel)}
}

// Set stores an active tunnel for the given port.
func (s *ActiveTunnelStore) Set(port int, t ActiveTunnel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tunnels[port] = t
}

// Get returns the active tunnel for a port, if any.
func (s *ActiveTunnelStore) Get(port int) (ActiveTunnel, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tunnels[port]
	return t, ok
}

// Delete removes the tunnel for a port.
func (s *ActiveTunnelStore) Delete(port int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tunnels, port)
}

// All returns all active tunnels.
func (s *ActiveTunnelStore) All() []ActiveTunnel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ActiveTunnel, 0, len(s.tunnels))
	for _, t := range s.tunnels {
		out = append(out, t)
	}
	return out
}
