package tools

import (
	"context"
	"fmt"
	"strings"
)

// ReviewUpstreamChangesTool shows upstream commits and diffs.
type ReviewUpstreamChangesTool struct{}

func (ReviewUpstreamChangesTool) Name() string        { return "review_upstream_changes" }
func (ReviewUpstreamChangesTool) Description() string { return "ALWAYS call before pull_upstream. Shows upstream commits with diffs. Use pull_upstream with commit hash to cherry-pick." }
func (ReviewUpstreamChangesTool) Parameters() string {
	return `{"type":"object","properties":{},"required":[]}`
}

func (ReviewUpstreamChangesTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	dir := resolveRepoPath("")
	// git fetch origin
	if _, err := runGit(ctx, dir, "fetch", "origin"); err != nil {
		return "Git fetch failed: " + err.Error(), nil
	}
	// git log HEAD..origin/main --oneline
	out, err := runGit(ctx, dir, "log", "HEAD..origin/main", "--oneline")
	if err != nil {
		return "Git log failed (maybe no origin/main): " + err.Error(), nil
	}
	if out == "" {
		return "Already up to date with origin/main.", nil
	}
	lines := strings.Split(out, "\n")
	// Get full diff for each commit
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d upstream commit(s) to review.\n\n", len(lines)))
	for i, line := range lines {
		parts := strings.SplitN(line, " ", 2)
		hash := ""
		msg := ""
		if len(parts) >= 1 {
			hash = parts[0]
		}
		if len(parts) >= 2 {
			msg = parts[1]
		}
		diff, _ := runGit(ctx, dir, "show", hash, "--stat")
		if len(diff) > 2000 {
			diff = diff[:2000] + "\n... (truncated)"
		}
		sb.WriteString(fmt.Sprintf("--- COMMIT %d/%d ---\nHash: %s\nMessage: %s\n\n%s\n--- END COMMIT ---\n\n", i+1, len(lines), hash, msg, diff))
	}
	return sb.String(), nil
}
