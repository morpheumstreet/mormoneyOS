package tools

import (
	"context"
	"strings"
)

// PullUpstreamTool applies upstream changes.
type PullUpstreamTool struct{}

func (PullUpstreamTool) Name() string        { return "pull_upstream" }
func (PullUpstreamTool) Description() string { return "Apply upstream changes. Call review_upstream_changes first. Use commit hash to cherry-pick, or omit to pull all." }
func (PullUpstreamTool) Parameters() string {
	return `{"type":"object","properties":{"commit":{"type":"string","description":"Commit hash to cherry-pick (preferred). Omit to pull all."}},"required":[]}`
}

func (PullUpstreamTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	commit, _ := args["commit"].(string)
	commit = strings.TrimSpace(commit)
	dir := resolveRepoPath("")

	if commit != "" {
		out, err := runGit(ctx, dir, "cherry-pick", commit)
		if err != nil {
			return "Cherry-pick failed: " + out, nil
		}
		return "Cherry-picked " + commit, nil
	}
	out, err := runGit(ctx, dir, "pull", "origin", "main", "--ff-only")
	if err != nil {
		return "Pull failed: " + out, nil
	}
	return "Pulled origin/main (fast-forward). " + out, nil
}
