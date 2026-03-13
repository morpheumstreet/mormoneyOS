package tools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func resolveRepoPath(path string) string {
	if path == "" || path == "~/.automaton" {
		return "."
	}
	if strings.HasPrefix(path, "~/") {
		if h, err := os.UserHomeDir(); err == nil && h != "" {
			return filepath.Join(h, path[2:])
		}
	}
	return path
}
