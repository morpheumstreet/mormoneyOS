package tools

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

// InstalledToolAdapter implements Tool for DB-installed tools (extension point, TS-aligned).
type InstalledToolAdapter struct {
	Tool state.InstalledTool
}

func (i *InstalledToolAdapter) Name() string { return i.Tool.Name }
func (i *InstalledToolAdapter) Description() string {
	return "Installed tool: " + i.Tool.Name
}
func (i *InstalledToolAdapter) Parameters() string {
	var cfg map[string]any
	if i.Tool.Config != "" {
		_ = json.Unmarshal([]byte(i.Tool.Config), &cfg)
	}
	if cfg != nil {
		if p, ok := cfg["parameters"].(map[string]any); ok {
			b, _ := json.Marshal(p)
			return string(b)
		}
	}
	return `{"type":"object","properties":{},"required":[]}`
}

func (i *InstalledToolAdapter) Execute(ctx context.Context, args map[string]any) (string, error) {
	switch i.Tool.Type {
	case "mcp":
		argsJSON, _ := json.Marshal(args)
		return "MCP tool " + i.Tool.Name + " invoked with args: " + string(argsJSON), nil
	}
	// custom / builtin: check for command in config
	var cfg map[string]any
	if i.Tool.Config != "" {
		_ = json.Unmarshal([]byte(i.Tool.Config), &cfg)
	}
	if cfg != nil {
		if cmd, ok := cfg["command"].(string); ok && cmd != "" {
			argsJSON, _ := json.Marshal(args)
			cmdStr := strings.TrimSpace(cmd) + " " + escapeShellArg(string(argsJSON))
			execCmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
			out, err := execCmd.CombinedOutput()
			if err != nil {
				return string(out) + "\nerror: " + err.Error(), nil
			}
			return string(out), nil
		}
	}
	return "Installed tool " + i.Tool.Name + " has no executable command configured.", nil
}

// InstalledToolsFromDB converts DB InstalledTool records to Tool implementations.
func InstalledToolsFromDB(tools []state.InstalledTool) []Tool {
	out := make([]Tool, 0, len(tools))
	for i := range tools {
		out = append(out, &InstalledToolAdapter{Tool: tools[i]})
	}
	return out
}
