package tunnel

import (
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// CloudflareProvider exposes localhost via Cloudflare Tunnel (cloudflared).
// Requires token from cloudflared tunnel create. Free tier.
func CloudflareProvider(pc types.TunnelProviderConfig) TunnelProvider {
	if pc.Token == "" {
		return nil
	}
	return NewCommandTunnelProviderWithReplacements(
		"cloudflare",
		"cloudflared tunnel --no-autoupdate run --token {token} --url http://{host}:{port}",
		"https://",
		"cloudflared",
		nil,
		map[string]string{"token": pc.Token},
	)
}
