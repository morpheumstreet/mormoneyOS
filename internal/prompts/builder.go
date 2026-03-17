package prompts

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

// FormatHistoryForReAct formats recent turns into a string for the react_cot template.
// Used when building a single combined user message (alternative to multi-turn format).
func FormatHistoryForReAct(turns []state.Turn) string {
	if len(turns) == 0 {
		return "(no prior turns)"
	}
	var b strings.Builder
	for i := len(turns) - 1; i >= 0; i-- {
		t := turns[i]
		if t.Input != "" {
			b.WriteString("User: ")
			b.WriteString(t.Input)
			b.WriteString("\n")
		}
		if t.Thinking != "" {
			b.WriteString("Assistant: ")
			b.WriteString(t.Thinking)
			if t.ToolCalls != "" {
				b.WriteString(" [tool results omitted]")
			}
			b.WriteString("\n")
		}
	}
	return strings.TrimSpace(b.String())
}

// RenderReactCoT renders the react_cot template with the given data.
// Used for the combined-context format (memory + history + input + instructions in one message).
func RenderReactCoT(memoryBlock, history, currentInput string) (string, error) {
	tmpls, err := loadV1Templates()
	if err != nil {
		return "", fmt.Errorf("load templates: %w", err)
	}
	data := map[string]string{
		"MemoryBlock":  memoryBlock,
		"History":      history,
		"CurrentInput": currentInput,
	}
	var buf bytes.Buffer
	if err := tmpls.ExecuteTemplate(&buf, "react_cot", data); err != nil {
		return "", fmt.Errorf("render react_cot: %w", err)
	}
	return buf.String(), nil
}
