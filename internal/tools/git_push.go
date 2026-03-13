package tools

import (
	"context"
	"strings"
)

// GitPushTool pushes to a git remote.
type GitPushTool struct{}

func (GitPushTool) Name() string        { return "git_push" }
func (GitPushTool) Description() string { return "Push to a git remote." }
func (GitPushTool) Parameters() string {
	return `{"type":"object","properties":{"path":{"type":"string","description":"Repository path"},"remote":{"type":"string","description":"Remote name (default: origin)"},"branch":{"type":"string","description":"Branch name (optional)"}},"required":["path"]}`
}

func (GitPushTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	remote, _ := args["remote"].(string)
	branch, _ := args["branch"].(string)
	path = strings.TrimSpace(path)
	if path == "" {
		return "", ErrInvalidArgs{Msg: "path required"}
	}
	if remote == "" {
		remote = "origin"
	}
	dir := resolveRepoPath(path)
	gitArgs := []string{"push", remote}
	if branch != "" {
		gitArgs = append(gitArgs, branch)
	}
	return runGit(ctx, dir, gitArgs...)
}
