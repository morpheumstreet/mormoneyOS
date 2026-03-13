package tools

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// ConfigTool implements Tool from config-driven definitions (extension point).
type ConfigTool struct {
	Def types.ConfigToolDef
}

func (c *ConfigTool) Name() string        { return c.Def.Name }
func (c *ConfigTool) Description() string { return c.Def.Description }
func (c *ConfigTool) Parameters() string {
	if c.Def.Parameters != "" {
		return c.Def.Parameters
	}
	return `{"type":"object","properties":{},"required":[]}`
}

func (c *ConfigTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if c.Def.Command == "" {
		return "Config tool " + c.Def.Name + " has no command configured.", nil
	}
	argsJSON, _ := json.Marshal(args)
	cmdStr := strings.TrimSpace(c.Def.Command) + " " + escapeShellArg(string(argsJSON))
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out) + "\nerror: " + err.Error(), nil
	}
	return string(out), nil
}

// ConfigToolsFromDefs converts config tool definitions to Tool implementations.
func ConfigToolsFromDefs(defs []types.ConfigToolDef) []Tool {
	out := make([]Tool, 0, len(defs))
	for i := range defs {
		out = append(out, &ConfigTool{Def: defs[i]})
	}
	return out
}
