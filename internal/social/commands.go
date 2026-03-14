package social

import "strings"

// IsCommand returns true if content looks like a slash command (e.g. /status, /help).
func IsCommand(content string) bool {
	s := strings.TrimSpace(content)
	return len(s) >= 2 && s[0] == '/' && s[1] != '/'
}

// ParseCommand extracts command name and optional args. E.g. "/status" -> ("status", ""), "/think high" -> ("think", "high").
func ParseCommand(content string) (cmd, args string) {
	s := strings.TrimSpace(content)
	if len(s) < 2 || s[0] != '/' {
		return "", ""
	}
	s = s[1:]
	idx := strings.IndexFunc(s, func(r rune) bool { return r == ' ' || r == '\t' })
	if idx < 0 {
		return strings.ToLower(s), ""
	}
	return strings.ToLower(s[:idx]), strings.TrimSpace(s[idx+1:])
}
