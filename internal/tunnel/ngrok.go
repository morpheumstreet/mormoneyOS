package tunnel

import (
	"fmt"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// NgrokProvider exposes localhost via ngrok.
// Requires authToken from ngrok dashboard. Optional domain for reserved subdomains.
func NgrokProvider(pc types.TunnelProviderConfig) TunnelProvider {
	if pc.AuthToken == "" {
		return nil
	}
	cmd := "ngrok http {port}"
	if pc.Domain != "" {
		cmd = fmt.Sprintf("ngrok http --domain=%s {port}", pc.Domain)
	}
	return NewCommandTunnelProviderWithReplacements(
		"ngrok",
		cmd,
		"https://",
		"ngrok",
		[]string{fmt.Sprintf("NGROK_AUTHTOKEN=%s", pc.AuthToken)},
		nil,
	)
}
