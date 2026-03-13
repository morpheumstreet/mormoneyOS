package tools

import (
	"context"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var npmPkgRegex = regexp.MustCompile(`^[@a-zA-Z0-9._\/-]+$`)

// InstallNpmPackageTool installs an npm package.
type InstallNpmPackageTool struct{}

func (InstallNpmPackageTool) Name() string        { return "install_npm_package" }
func (InstallNpmPackageTool) Description() string { return "Install an npm package in your environment." }
func (InstallNpmPackageTool) Parameters() string {
	return `{"type":"object","properties":{"package":{"type":"string","description":"Package name (e.g., axios)"}},"required":["package"]}`
}

func (InstallNpmPackageTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	pkg, _ := args["package"].(string)
	pkg = strings.TrimSpace(pkg)
	if pkg == "" {
		return "", ErrInvalidArgs{Msg: "package required"}
	}
	if !npmPkgRegex.MatchString(pkg) {
		return "Blocked: invalid package name \"" + pkg + "\"", nil
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "npm", "install", "-g", pkg)
	out, err := cmd.CombinedOutput()
	result := strings.TrimSpace(string(out))
	if err != nil {
		return "Failed to install " + pkg + ": " + result, nil
	}
	return "Installed: " + pkg, nil
}
