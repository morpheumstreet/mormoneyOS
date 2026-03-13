package tools

import (
	"context"
	"fmt"
	"strings"
)

const maxProcedureLen = 10000
const procedurePrefix = "procedure:"

// SaveProcedureTool saves a procedure by name.
type SaveProcedureTool struct {
	Store ToolStore
}

func (SaveProcedureTool) Name() string        { return "save_procedure" }
func (SaveProcedureTool) Description() string { return "Save a procedure for later recall." }
func (SaveProcedureTool) Parameters() string {
	return `{"type":"object","properties":{"name":{"type":"string","description":"Procedure name"},"steps":{"type":"string","description":"Procedure steps"}},"required":["name","steps"]}`
}

func (t *SaveProcedureTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "save_procedure requires store"}
	}
	name, _ := args["name"].(string)
	steps, _ := args["steps"].(string)
	name = strings.TrimSpace(name)
	steps = strings.TrimSpace(steps)
	if name == "" || steps == "" {
		return "", ErrInvalidArgs{Msg: "name and steps required"}
	}
	if len(steps) > maxProcedureLen {
		return fmt.Sprintf("Error: Procedure exceeds %d character limit", maxProcedureLen), nil
	}
	key := procedurePrefix + name
	if err := t.Store.SetKV(key, steps); err != nil {
		return "", err
	}
	return fmt.Sprintf("Procedure '%s' saved.", name), nil
}

// RecallProcedureTool recalls a procedure by name.
type RecallProcedureTool struct {
	Store ToolStore
}

func (RecallProcedureTool) Name() string        { return "recall_procedure" }
func (RecallProcedureTool) Description() string { return "Recall a saved procedure." }
func (RecallProcedureTool) Parameters() string {
	return `{"type":"object","properties":{"name":{"type":"string","description":"Procedure name"}},"required":["name"]}`
}

func (t *RecallProcedureTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "recall_procedure requires store"}
	}
	name, _ := args["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrInvalidArgs{Msg: "name required"}
	}
	key := procedurePrefix + name
	s, ok, _ := t.Store.GetKV(key)
	if !ok || s == "" {
		return "Procedure not found.", nil
	}
	return s, nil
}
