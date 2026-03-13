package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/morpheumlabs/mormoneyos-go/internal/tunnel"
)

// TunnelStatusTool lists active tunnels.
type TunnelStatusTool struct {
	Manager *tunnel.TunnelManager
}

func (t *TunnelStatusTool) Name() string        { return "tunnel_status" }
func (t *TunnelStatusTool) Description() string { return "List all active tunnels (exposed ports)." }

func (t *TunnelStatusTool) Parameters() string {
	return `{"type":"object","properties":{},"required":[]}`
}

func (t *TunnelStatusTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Manager == nil {
		return "", fmt.Errorf("tunnel not configured")
	}
	active := t.Manager.Status()
	if len(active) == 0 {
		return `{"tunnels":[]}`, nil
	}
	list := make([]map[string]any, len(active))
	for i, a := range active {
		list[i] = map[string]any{"port": a.Port, "provider": a.Provider, "public_url": a.PublicURL}
	}
	out := map[string]any{"tunnels": list}
	b, _ := json.Marshal(out)
	return string(b), nil
}
