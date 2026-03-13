package tools

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// ShellTool executes shell commands. Policy must allow before invocation.
type ShellTool struct{}

func (ShellTool) Name() string        { return "shell" }
func (ShellTool) Description() string { return "Execute a shell command. Use for listing files, running scripts, or system queries. Never run destructive commands." }
func (ShellTool) Parameters() string { return `{"type":"object","properties":{"command":{"type":"string","description":"The shell command to run"}},"required":["command"]}` }

func (ShellTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	cmdStr, _ := args["command"].(string)
	if cmdStr == "" {
		cmdStr, _ = args["cmd"].(string)
	}
	cmdStr = strings.TrimSpace(cmdStr)
	if cmdStr == "" {
		return "", ErrInvalidArgs{Msg: "command or cmd required"}
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	out, err := cmd.CombinedOutput()
	result := strings.TrimSpace(string(out))
	if err != nil {
		return result, err
	}
	return result, nil
}

// ErrInvalidArgs for malformed tool arguments.
type ErrInvalidArgs struct {
	Msg string
}

func (e ErrInvalidArgs) Error() string {
	return "invalid args: " + e.Msg
}
