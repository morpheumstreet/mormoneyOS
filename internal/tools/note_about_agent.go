package tools

import (
	"context"
	"fmt"
	"strings"
)

const agentNotePrefix = "agent_note:"
const maxNoteLen = 1000

// NoteAboutAgentTool stores a note about another agent by address.
type NoteAboutAgentTool struct {
	Store ToolStore
}

func (NoteAboutAgentTool) Name() string        { return "note_about_agent" }
func (NoteAboutAgentTool) Description() string { return "Store a note about another agent." }
func (NoteAboutAgentTool) Parameters() string {
	return `{"type":"object","properties":{"address":{"type":"string","description":"Agent address"},"note":{"type":"string","description":"Note content"}},"required":["address","note"]}`
}

func (t *NoteAboutAgentTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "note_about_agent requires store"}
	}
	addr, _ := args["address"].(string)
	note, _ := args["note"].(string)
	addr = strings.TrimSpace(addr)
	note = strings.TrimSpace(note)
	if addr == "" || note == "" {
		return "", ErrInvalidArgs{Msg: "address and note required"}
	}
	if len(note) > maxNoteLen {
		return fmt.Sprintf("Error: Note exceeds %d character limit", maxNoteLen), nil
	}
	key := agentNotePrefix + addr
	if err := t.Store.SetKV(key, note); err != nil {
		return "", err
	}
	return "Note saved.", nil
}
