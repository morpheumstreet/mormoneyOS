package skills

import (
	"strings"
)

// FormatForPrompt formats skills for injection into the system prompt.
// Truncates instructions when over tokenBudgetMax chars (default 2000).
func FormatForPrompt(skills []*Skill, tokenBudgetMax int) string {
	if len(skills) == 0 {
		return ""
	}
	if tokenBudgetMax <= 0 {
		tokenBudgetMax = 2000
	}
	var b strings.Builder
	b.WriteString("\n--- ENABLED SKILLS ---\n")
	remaining := tokenBudgetMax
	for _, s := range skills {
		header := s.Name + ": " + s.Description + "\n"
		if len(header) > remaining {
			break
		}
		b.WriteString(header)
		remaining -= len(header)
		inst := s.Instructions
		if len(inst) > remaining {
			inst = inst[:remaining] + "..."
			remaining = 0
		} else {
			remaining -= len(inst)
		}
		if inst != "" {
			b.WriteString(inst)
			b.WriteString("\n\n")
		}
		if remaining <= 0 {
			break
		}
	}
	b.WriteString("--- END SKILLS ---\n")
	return b.String()
}
