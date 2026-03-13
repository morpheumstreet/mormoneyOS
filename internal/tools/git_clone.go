package tools

import (
	"context"
	"strconv"
	"strings"
)

// GitCloneTool clones a git repository.
type GitCloneTool struct{}

func (GitCloneTool) Name() string        { return "git_clone" }
func (GitCloneTool) Description() string { return "Clone a git repository." }
func (GitCloneTool) Parameters() string {
	return `{"type":"object","properties":{"url":{"type":"string","description":"Repository URL"},"path":{"type":"string","description":"Target directory"},"depth":{"type":"number","description":"Shallow clone depth (optional)"}},"required":["url","path"]}`
}

func (GitCloneTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	url, _ := args["url"].(string)
	path, _ := args["path"].(string)
	depth, _ := args["depth"].(float64)
	url = strings.TrimSpace(url)
	path = strings.TrimSpace(path)
	if url == "" || path == "" {
		return "", ErrInvalidArgs{Msg: "url and path required"}
	}
	gitArgs := []string{"clone", url, path}
	if depth > 0 {
		gitArgs = append(gitArgs, "--depth", strconv.Itoa(int(depth)))
	}
	return runGit(ctx, ".", gitArgs...)
}
