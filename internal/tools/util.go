package tools

import "strings"

// escapeShellArg quotes a string for safe use in shell commands.
func escapeShellArg(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
