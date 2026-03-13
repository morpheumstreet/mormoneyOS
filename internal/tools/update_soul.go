package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const maxSoulLen = 5000

// UpdateSoulTool updates the soul document in KV.
type UpdateSoulTool struct {
	Store ToolStore
}

func (UpdateSoulTool) Name() string        { return "update_soul" }
func (UpdateSoulTool) Description() string { return "Update your soul document (identity, values, constraints)." }
func (UpdateSoulTool) Parameters() string {
	return `{"type":"object","properties":{"content":{"type":"string","description":"New soul content"}},"required":["content"]}`
}

func (t *UpdateSoulTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "update_soul requires store"}
	}
	content, _ := args["content"].(string)
	content = strings.TrimSpace(content)
	if content == "" {
		return "", ErrInvalidArgs{Msg: "content required"}
	}
	if len(content) > maxSoulLen {
		return fmt.Sprintf("Error: Soul content exceeds %d character limit (%d chars)", maxSoulLen, len(content)), nil
	}
	old, _, _ := t.Store.GetKV("soul_content")
	if old != "" {
		_ = t.Store.SetKV("soul_content_backup", old)
		// Append to soul history
		raw, _, _ := t.Store.GetKV(soulHistoryKey)
		var hist []soulHistoryEntry
		if raw != "" {
			_ = json.Unmarshal([]byte(raw), &hist)
		}
		preview := old
		if len(preview) > 80 {
			preview = preview[:80] + "..."
		}
		hist = append([]soulHistoryEntry{{At: time.Now().Format(time.RFC3339), Preview: preview}}, hist...)
		if len(hist) > maxHistoryEntries {
			hist = hist[:maxHistoryEntries]
		}
		if b, err := json.Marshal(hist); err == nil {
			_ = t.Store.SetKV(soulHistoryKey, string(b))
		}
	}
	if err := t.Store.SetKV("soul_content", content); err != nil {
		return "", err
	}
	return fmt.Sprintf("Soul updated (%d chars). Previous version backed up.", len(content)), nil
}
