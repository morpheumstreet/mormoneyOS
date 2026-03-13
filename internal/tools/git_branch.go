package tools

import (
	"context"
	"strings"
)

// GitBranchTool manages git branches.
type GitBranchTool struct{}

func (GitBranchTool) Name() string        { return "git_branch" }
func (GitBranchTool) Description() string { return "Manage git branches (list, create, checkout, delete)." }
func (GitBranchTool) Parameters() string {
	return `{"type":"object","properties":{"path":{"type":"string","description":"Repository path"},"action":{"type":"string","description":"list, create, checkout, or delete"},"branch_name":{"type":"string","description":"Branch name (for create/checkout/delete)"}},"required":["path","action"]}`
}

func (GitBranchTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	action, _ := args["action"].(string)
	branch, _ := args["branch_name"].(string)
	path = strings.TrimSpace(path)
	action = strings.ToLower(strings.TrimSpace(action))
	if path == "" || action == "" {
		return "", ErrInvalidArgs{Msg: "path and action required"}
	}
	dir := resolveRepoPath(path)
	switch action {
	case "list":
		return runGit(ctx, dir, "branch", "-a")
	case "create":
		if branch == "" {
			return "", ErrInvalidArgs{Msg: "branch_name required for create"}
		}
		return runGit(ctx, dir, "checkout", "-b", branch)
	case "checkout":
		if branch == "" {
			return "", ErrInvalidArgs{Msg: "branch_name required for checkout"}
		}
		return runGit(ctx, dir, "checkout", branch)
	case "delete":
		if branch == "" {
			return "", ErrInvalidArgs{Msg: "branch_name required for delete"}
		}
		return runGit(ctx, dir, "branch", "-d", branch)
	default:
		return "", ErrInvalidArgs{Msg: "action must be list, create, checkout, or delete"}
	}
}
