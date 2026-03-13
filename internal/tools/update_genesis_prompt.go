package tools

import (
	"context"
	"fmt"
	"strings"
)

const maxGenesisPromptLen = 2000

// UpdateGenesisPromptTool updates the genesis prompt in KV.
type UpdateGenesisPromptTool struct {
	Store ToolStore
}

func (UpdateGenesisPromptTool) Name() string        { return "update_genesis_prompt" }
func (UpdateGenesisPromptTool) Description() string { return "Update your genesis prompt. Requires strong justification." }
func (UpdateGenesisPromptTool) Parameters() string {
	return `{"type":"object","properties":{"new_prompt":{"type":"string","description":"New genesis prompt"},"reason":{"type":"string","description":"Why you are changing it"}},"required":["new_prompt","reason"]}`
}

func (t *UpdateGenesisPromptTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "update_genesis_prompt requires store"}
	}
	prompt, _ := args["new_prompt"].(string)
	reason, _ := args["reason"].(string)
	prompt = strings.TrimSpace(prompt)
	reason = strings.TrimSpace(reason)
	if prompt == "" || reason == "" {
		return "", ErrInvalidArgs{Msg: "new_prompt and reason required"}
	}
	if len(prompt) > maxGenesisPromptLen {
		return fmt.Sprintf("Error: Genesis prompt exceeds %d character limit (%d chars)", maxGenesisPromptLen, len(prompt)), nil
	}
	old, _, _ := t.Store.GetKV("genesis_prompt")
	if old != "" {
		_ = t.Store.SetKV("genesis_prompt_backup", old)
	}
	_ = t.Store.SetKV("genesis_prompt", prompt)
	return fmt.Sprintf("Genesis prompt updated (%d chars). Reason: %s. Previous version backed up.", len(prompt), reason), nil
}
