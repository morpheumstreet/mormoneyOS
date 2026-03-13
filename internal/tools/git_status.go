package tools

import (
	"context"
)

// GitStatusTool shows git status.
type GitStatusTool struct{}

func (GitStatusTool) Name() string        { return "git_status" }
func (GitStatusTool) Description() string { return "Show git status for a repository." }
func (GitStatusTool) Parameters() string {
	return `{"type":"object","properties":{"path":{"type":"string","description":"Repository path (default: .)"}},"required":[]}`
}

func (GitStatusTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	dir := resolveRepoPath(path)
	out, err := runGit(ctx, dir, "status", "--short", "-b")
	if err != nil {
		return out, err
	}
	if out == "" {
		return "Clean working tree.", nil
	}
	return out, nil
}
