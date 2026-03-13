package agent

import (
	"encoding/json"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/inference"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

// BuildContextMessages builds the message array for inference (TS buildContextMessages-aligned, simplified).
// Returns: [system, user (input), assistant (thinking + tool results), ...] for each recent turn, then current user input.
func BuildContextMessages(
	systemPrompt string,
	recentTurns []state.Turn,
	pendingInput string,
) []inference.ChatMessage {
	msgs := []inference.ChatMessage{
		{Role: "system", Content: systemPrompt},
	}

	for _, t := range recentTurns {
		if t.Input != "" {
			msgs = append(msgs, inference.ChatMessage{Role: "user", Content: t.Input})
		}
		if t.Thinking != "" || t.ToolCalls != "" {
			content := t.Thinking
			if t.ToolCalls != "" {
				content = appendToolResults(content, t.ToolCalls)
			}
			msgs = append(msgs, inference.ChatMessage{Role: "assistant", Content: content})
		}
	}

	if pendingInput != "" {
		msgs = append(msgs, inference.ChatMessage{Role: "user", Content: pendingInput})
	}

	return msgs
}

func appendToolResults(thinking, toolCallsJSON string) string {
	var tcList []struct {
		Name   string `json:"name"`
		Result string `json:"result"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal([]byte(toolCallsJSON), &tcList); err != nil || len(tcList) == 0 {
		return thinking
	}
	var b strings.Builder
	if thinking != "" {
		b.WriteString(thinking)
		b.WriteString("\n\n")
	}
	b.WriteString("[Tool results:\n")
	for _, tc := range tcList {
		b.WriteString("- ")
		b.WriteString(tc.Name)
		b.WriteString(": ")
		if tc.Error != "" {
			b.WriteString("error: ")
			b.WriteString(tc.Error)
		} else {
			b.WriteString(tc.Result)
		}
		b.WriteString("\n")
	}
	b.WriteString("]")
	return b.String()
}
