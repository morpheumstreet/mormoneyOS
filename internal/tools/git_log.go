package tools

import (
	"context"
	"strconv"
)

// GitLogTool shows git commit history.
type GitLogTool struct{}

func (GitLogTool) Name() string        { return "git_log" }
func (GitLogTool) Description() string { return "View git commit history." }
func (GitLogTool) Parameters() string {
	return `{"type":"object","properties":{"path":{"type":"string","description":"Repository path"},"limit":{"type":"number","description":"Number of commits (default: 10)"}},"required":[]}`
}

func (GitLogTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	dir := resolveRepoPath(path)
	out, err := runGit(ctx, dir, "log", "-n", strconv.Itoa(limit), "--oneline")
	if err != nil {
		return "", err
	}
	if out == "" {
		return "No commits yet.", nil
	}
	return out, nil
}
