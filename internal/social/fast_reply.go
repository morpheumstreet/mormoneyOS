package social

import (
	"regexp"
	"strings"
)

// ClassifyFastReply determines if a message should get a fast (programmatic) reply.
// Returns (fast, normalizedContent). When fast is true, normalizedContent is the
// content to pass to the handler (e.g. "/status" for both "status" and "/status").
// Zero LLM, pure rule-based. Used by check_social_inbox before any inbox insertion.
func ClassifyFastReply(content string) (fast bool, normalizedContent string) {
	s := strings.TrimSpace(content)
	if s == "" {
		return false, ""
	}
	lower := strings.ToLower(s)

	// 1. Slash commands — pass through as-is
	if len(s) >= 2 && s[0] == '/' && s[1] != '/' {
		return true, s
	}

	// 2. !ping, ping, pong — instant pong
	if matchExact(lower, "ping", "!ping", "pong") {
		return true, "/ping"
	}

	// 3. Status aliases
	if matchExact(lower, "status", "st") {
		return true, "/status"
	}

	// 4. Credits / balance aliases
	if matchExact(lower, "credits", "credits?", "balance", "balances", "usdc") {
		return true, "/balance"
	}

	// 5. Help
	if matchExact(lower, "help", "?") {
		return true, "/help"
	}

	// 6. Uptime — maps to status
	if matchExact(lower, "uptime") {
		return true, "/status"
	}

	// 7. !cmd <subcommand> — e.g. !cmd balance, !cmd status
	if strings.HasPrefix(lower, "!cmd ") {
		sub := strings.TrimSpace(lower[5:])
		if sub != "" {
			// Map common subcommands
			switch sub {
			case "balance", "balances", "credits", "usdc":
				return true, "/balance"
			case "status", "st":
				return true, "/status"
			case "help":
				return true, "/help"
			case "skill", "skills":
				return true, "/skill"
			default:
				return true, "/" + sub
			}
		}
	}

	// 8. Short question patterns — "status?" or "credits?" (already covered above)
	// 9. @bot mention + command — strip bot mention, re-check (only if we actually stripped something)
	if stripped := stripBotMention(s); stripped != "" && stripped != s {
		return ClassifyFastReply(stripped)
	}

	return false, ""
}

// matchExact returns true if lower equals any of the options (case-insensitive).
func matchExact(lower string, options ...string) bool {
	for _, opt := range options {
		if lower == strings.ToLower(opt) {
			return true
		}
	}
	return false
}

// stripBotMention removes leading @username or @username: from content.
// Example: "@mybot status" -> "status", "@mybot: help" -> "help"
var botMentionRe = regexp.MustCompile(`^@\w+(?:\s*:\s*)?\s*`)

func stripBotMention(content string) string {
	return strings.TrimSpace(botMentionRe.ReplaceAllString(content, ""))
}
