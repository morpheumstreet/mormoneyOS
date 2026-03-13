package tools

import (
	"context"
	"strings"
)

// GitCommitTool creates a git commit.
type GitCommitTool struct{}

func (GitCommitTool) Name() string        { return "git_commit" }
func (GitCommitTool) Description() string { return "Create a git commit." }
func (GitCommitTool) Parameters() string {
	return `{"type":"object","properties":{"path":{"type":"string","description":"Repository path"},"message":{"type":"string","description":"Commit message"},"add_all":{"type":"boolean","description":"Stage all changes first (default: true)"}},"required":["message"]}`
}

func (GitCommitTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	msg, _ := args["message"].(string)
	addAll := true
	if a, ok := args["add_all"].(bool); ok {
		addAll = a
	}
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return "", ErrInvalidArgs{Msg: "message required"}
	}
	dir := resolveRepoPath(path)
	if addAll {
		if _, err := runGit(ctx, dir, "add", "-A"); err != nil {
			return "", err
		}
	}
	out, err := runGit(ctx, dir, "commit", "-m", msg)
	if err != nil {
		return out, err
	}
	return "Commit created.", nil
}
