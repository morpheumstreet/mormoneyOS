package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/morpheumlabs/mormoneyos-go/internal/tunnel"
)

// RemovePortTool stops the tunnel for a port.
type RemovePortTool struct {
	Manager *tunnel.TunnelManager
}

func (t *RemovePortTool) Name() string        { return "remove_port" }
func (t *RemovePortTool) Description() string { return "Stop the tunnel for a previously exposed port." }

func (t *RemovePortTool) Parameters() string {
	return `{"type":"object","properties":{"port":{"type":"integer","description":"Port whose tunnel to stop"}},"required":["port"]}`
}

func (t *RemovePortTool) Execute(ctx context.Context, args map[string]any) (string, error) {
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
	if err := t.Manager.Stop(port); err != nil {
		return "", err
	}
	out := map[string]any{"removed": true, "port": port}
	b, _ := json.Marshal(out)
	return string(b), nil
}
