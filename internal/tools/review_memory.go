package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ReviewMemoryTool summarizes stored memory (facts, goals, soul).
type ReviewMemoryTool struct {
	Store ToolStore
}

func (ReviewMemoryTool) Name() string        { return "review_memory" }
func (ReviewMemoryTool) Description() string { return "Review your stored memory (facts, goals, soul summary)." }
func (ReviewMemoryTool) Parameters() string {
	return `{"type":"object","properties":{},"required":[]}`
}

func (t *ReviewMemoryTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "No store configured.", nil
	}
	var parts []string

	// Facts
	if raw, ok, _ := t.Store.GetKV(memoryFactsKey); ok && raw != "" {
		var facts []memoryFact
		if json.Unmarshal([]byte(raw), &facts) == nil {
			parts = append(parts, fmt.Sprintf("Facts: %d stored", len(facts)))
		}
	}

	// Goals
	if raw, ok, _ := t.Store.GetKV(goalsKey); ok && raw != "" {
		var goals []goal
		if json.Unmarshal([]byte(raw), &goals) == nil {
			pending := 0
			for _, g := range goals {
				if g.DoneAt == "" {
					pending++
				}
			}
			parts = append(parts, fmt.Sprintf("Goals: %d total, %d pending", len(goals), pending))
		}
	}

	// Soul
	if s, ok, _ := t.Store.GetKV("soul_content"); ok && s != "" {
		preview := s
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		preview = strings.ReplaceAll(preview, "\n", " ")
		parts = append(parts, fmt.Sprintf("Soul: %d chars (%s)", len(s), preview))
	}

	if len(parts) == 0 {
		return "No memory stored. Use remember_fact, set_goal, or update_soul.", nil
	}
	return strings.Join(parts, "\n"), nil
}
