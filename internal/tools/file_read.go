package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// FileReadTool reads file contents. Policy must allow path before invocation.
type FileReadTool struct{}

func (FileReadTool) Name() string        { return "file_read" }
func (FileReadTool) Description() string { return "Read the contents of a file. Provide path (relative or absolute). Do not read sensitive paths." }
func (FileReadTool) Parameters() string { return `{"type":"object","properties":{"path":{"type":"string","description":"Path to the file"}},"required":["path"]}` }

func (FileReadTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	if path == "" {
		path, _ = args["file_path"].(string)
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return "", ErrInvalidArgs{Msg: "path or file_path required"}
	}

	// Resolve to absolute; reject path traversal
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	// Reject explicit traversal attempts
	if strings.Contains(path, "..") {
		return "", ErrInvalidArgs{Msg: "path traversal not allowed"}
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
