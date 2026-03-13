package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// FileWriteTool writes content to a file. Policy must allow path before invocation.
type FileWriteTool struct{}

func (FileWriteTool) Name() string { return "write_file" }
func (FileWriteTool) Description() string {
	return "Write content to a file. Provide path and content. Do not write to sensitive paths."
}
func (FileWriteTool) Parameters() string {
	return `{"type":"object","properties":{"path":{"type":"string","description":"Path to the file"},"content":{"type":"string","description":"Content to write"}},"required":["path","content"]}`
}

func (FileWriteTool) Execute(ctx context.Context, args map[string]any) (string, error) {
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
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, []byte(content), 0644); err != nil {
		return "", err
	}
	return "File written successfully.", nil
}
