package tunnel

import "context"

// TunnelProvider is implemented by bore, cloudflare, ngrok, custom, etc.
// External packages can implement and register new providers.
type TunnelProvider interface {
	Name() string
	Start(ctx context.Context, host string, port int) (publicURL string, err error)
	Stop(port int) error
	IsAvailable() bool
}

// ActiveTunnel represents a running tunnel.
type ActiveTunnel struct {
	Port      int
	Provider  string
	PublicURL string
}
