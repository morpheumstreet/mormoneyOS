package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/morpheumlabs/mormoneyos-go/internal/tunnel"
)

// ExposePortTool exposes a local port via a tunnel provider.
type ExposePortTool struct {
	Manager  *tunnel.TunnelManager
	Registry *tunnel.ProviderRegistry
	Default  string
}

func (t *ExposePortTool) Name() string        { return "expose_port" }
func (t *ExposePortTool) Description() string { return "Expose a local port to the internet via a tunnel. Returns the public URL. Prefer bore or localtunnel for cost-effectiveness." }

func (t *ExposePortTool) Parameters() string {
	providers := []string{"bore"}
	if t.Registry != nil {
		providers = t.Registry.List()
	}
	if len(providers) == 0 {
		providers = []string{"bore"}
	}
	enumJSON, _ := json.Marshal(providers)
	return fmt.Sprintf(`{"type":"object","properties":{"port":{"type":"integer","description":"Local port to expose (e.g. 8080 for web dashboard)"},"provider":{"type":"string","enum":%s,"description":"Tunnel provider"},"host":{"type":"string","description":"Local host (default: 127.0.0.1)"}},"required":["port"]}`, string(enumJSON))
}

func (t *ExposePortTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Manager == nil {
		return "", fmt.Errorf("tunnel not configured")
	}
	portVal, ok := args["port"]
	if !ok {
		return "", fmt.Errorf("port is required")
	}
	port := 0
	switch v := portVal.(type) {
	case float64:
		port = int(v)
	case int:
		port = v
	case string:
		var err error
		port, err = strconv.Atoi(v)
		if err != nil {
			return "", fmt.Errorf("port must be a number")
		}
	default:
		return "", fmt.Errorf("port must be a number")
	}
	if port <= 0 || port > 65535 {
		return "", fmt.Errorf("port must be 1-65535")
	}
	provider, _ := args["provider"].(string)
	if provider == "" {
		provider = t.Default
	}
	if provider == "" {
		provider = "bore"
	}
	host, _ := args["host"].(string)
	if host == "" {
		host = "127.0.0.1"
	}
	publicURL, err := t.Manager.Start(ctx, provider, host, port)
	if err != nil {
		return "", err
	}
	out := map[string]any{"public_url": publicURL, "provider": provider, "port": port}
	b, _ := json.Marshal(out)
	return string(b), nil
}
