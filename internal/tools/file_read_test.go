package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileReadTool_Execute(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(fpath, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	tool := FileReadTool{}

	out, err := tool.Execute(ctx, map[string]any{"path": fpath})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out != "hello world" {
		t.Errorf("got %q, want hello world", out)
	}
}

func TestFileReadTool_Execute_FileNotFound(t *testing.T) {
	ctx := context.Background()
	tool := FileReadTool{}

	_, err := tool.Execute(ctx, map[string]any{"path": "/nonexistent/file"})
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
