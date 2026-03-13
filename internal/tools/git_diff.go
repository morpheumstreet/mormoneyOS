package tools

import (
	"context"
)

// GitDiffTool shows git diff.
type GitDiffTool struct{}

func (GitDiffTool) Name() string        { return "git_diff" }
func (GitDiffTool) Description() string { return "Show git diff for a repository." }
func (GitDiffTool) Parameters() string {
	return `{"type":"object","properties":{"path":{"type":"string","description":"Repository path"},"staged":{"type":"boolean","description":"Show staged changes only"}},"required":[]}`
}

func (GitDiffTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	staged, _ := args["staged"].(bool)
	dir := resolveRepoPath(path)
	gitArgs := []string{"diff"}
	if staged {
		gitArgs = append(gitArgs, "--cached")
	}
	return runGit(ctx, dir, gitArgs...)
}
