package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileWriteTool_Execute(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	t.Run("writes file", func(t *testing.T) {
		path := filepath.Join(dir, "test.txt")
		tool := FileWriteTool{}
		result, err := tool.Execute(ctx, map[string]any{
			"path":    path,
			"content": "hello world",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "File written successfully." {
			t.Errorf("got %q", result)
		}
		data, _ := os.ReadFile(path)
		if string(data) != "hello world" {
			t.Errorf("got content %q", string(data))
		}
	})

	t.Run("rejects path traversal", func(t *testing.T) {
		tool := FileWriteTool{}
		_, err := tool.Execute(ctx, map[string]any{
			"path":    dir + "/../etc/passwd",
			"content": "x",
		})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("requires path", func(t *testing.T) {
		tool := FileWriteTool{}
		_, err := tool.Execute(ctx, map[string]any{"content": "x"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
