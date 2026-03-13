package tools

import (
	"context"
	"fmt"
)

// SystemSynopsisTool returns a system status report.
type SystemSynopsisTool struct {
	Store     ToolStore
	AppName   string
}

func (SystemSynopsisTool) Name() string { return "system_synopsis" }
func (SystemSynopsisTool) Description() string {
	return "Get a system status report: state, installed tools, heartbeat status, turn count."
}
func (SystemSynopsisTool) Parameters() string {
	return `{"type":"object","properties":{},"required":[]}`
}

func (t *SystemSynopsisTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "system_synopsis requires store"}
	}
	state, _, _ := t.Store.GetAgentState()
	if state == "" {
		state = "running"
	}
	turns, _ := t.Store.GetTurnCount()
	skills, ok := t.Store.GetSkills()
	skillCount := 0
	if ok {
		skillCount = len(skills)
	}
	name := t.AppName
	if name == "" {
		name = "moneyclaw"
	}
	return fmt.Sprintf(`=== SYSTEM SYNOPSIS ===
Name: %s
State: %s
Total turns: %d
Installed skills: %d
========================`, name, state, turns, skillCount), nil
}
