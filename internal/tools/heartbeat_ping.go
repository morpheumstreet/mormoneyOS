package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/morpheumlabs/mormoneyos-go/internal/conway"
)

// HeartbeatPingTool publishes a heartbeat status ping.
type HeartbeatPingTool struct {
	Conway conway.Client
	Store  ToolStore
}

func (HeartbeatPingTool) Name() string        { return "heartbeat_ping" }
func (HeartbeatPingTool) Description() string { return "Publish a heartbeat status ping. Shows the world you are alive." }
func (HeartbeatPingTool) Parameters() string {
	return `{"type":"object","properties":{},"required":[]}`
}

func (t *HeartbeatPingTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Conway == nil {
		return "", ErrConwayNotConfigured
	}
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "heartbeat_ping requires store"}
	}
	credits, _ := t.Conway.GetCreditsBalance(ctx)
	state, _, _ := t.Store.GetAgentState()
	if state == "" {
		state = "running"
	}
	startTime, _, _ := t.Store.GetKV("start_time")
	if startTime == "" {
		startTime = time.Now().UTC().Format(time.RFC3339)
		_ = t.Store.SetKV("start_time", startTime)
	}
	start, _ := time.Parse(time.RFC3339, startTime)
	uptimeSec := int64(time.Since(start).Seconds())
	_ = t.Store.SetKV("last_heartbeat_ping", fmt.Sprintf(`{"credits":%d,"state":"%s","uptime":%d}`, credits, state, uptimeSec))
	return fmt.Sprintf("Heartbeat published: %s | credits: $%.2f | uptime: %ds", state, float64(credits)/100, uptimeSec), nil
}
