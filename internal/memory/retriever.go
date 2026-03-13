package memory

import (
	"context"
	"fmt"
	"strings"
)

// MemoryRetriever retrieves relevant memories for context injection (TS step 6).
// Returns a formatted block or empty string. Errors must not block the agent loop.
type MemoryRetriever interface {
	Retrieve(ctx context.Context, sessionID string, currentInput string) (block string, err error)
}

// MemoryBlock holds sections for formatting (TS formatMemoryBlock-aligned).
// Phase 1: Known Facts, Active Goals, Known Procedures.
type MemoryBlock struct {
	Facts      []string // Known Facts (from memory_facts)
	Goals      []string // Active Goals (from goals, pending only)
	Procedures []ProcedureEntry
}

// ProcedureEntry is a procedure name and step count (Phase 1 format).
type ProcedureEntry struct {
	Name  string
	Steps int
}

// FormatMemoryBlock formats retrieval result into a markdown block for context.
// TS formatMemoryBlock-aligned. Returns "" when empty.
func FormatMemoryBlock(r *MemoryBlock) string {
	if r == nil {
		return ""
	}
	var sections []string
	if len(r.Facts) > 0 {
		lines := []string{"### Known Facts"}
		for _, f := range r.Facts {
			lines = append(lines, "- "+f)
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}
	if len(r.Goals) > 0 {
		lines := []string{"### Active Goals"}
		for _, g := range r.Goals {
			lines = append(lines, "- "+g)
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}
	if len(r.Procedures) > 0 {
		lines := []string{"### Known Procedures"}
		for _, p := range r.Procedures {
			lines = append(lines, "- "+p.Name+": "+formatSteps(p.Steps)+" steps")
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}
	if len(sections) == 0 {
		return ""
	}
	return "## Memory\n\n" + strings.Join(sections, "\n\n")
}

func formatSteps(n int) string {
	return fmt.Sprintf("%d", n)
}
