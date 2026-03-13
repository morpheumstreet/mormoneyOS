package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// EditOwnFileTool edits a file (full overwrite). Path protection via policy.
type EditOwnFileTool struct{}

func (EditOwnFileTool) Name() string        { return "edit_own_file" }
func (EditOwnFileTool) Description() string { return "Edit a file in your codebase. Full content replace. Some paths are protected." }
func (EditOwnFileTool) Parameters() string {
	return `{"type":"object","properties":{"path":{"type":"string","description":"File path"},"content":{"type":"string","description":"New content"},"description":{"type":"string","description":"Why you are making this change"}},"required":["path","content","description"]}`
}

func (EditOwnFileTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	path = strings.TrimSpace(path)
	if path == "" {
		return "", ErrInvalidArgs{Msg: "path required"}
	}
	if strings.Contains(path, "..") {
		return "", ErrInvalidArgs{Msg: "path traversal not allowed"}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(abs)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, []byte(content), 0644); err != nil {
		return "", err
	}
	return "File edited: " + path, nil
}
