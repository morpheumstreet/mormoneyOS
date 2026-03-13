package tools

import (
	"context"
	"testing"
)

func TestShellTool_Execute(t *testing.T) {
	ctx := context.Background()
	tool := ShellTool{}

	out, err := tool.Execute(ctx, map[string]any{"command": "echo hello"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out != "hello" {
		t.Errorf("got %q, want hello", out)
	}
}

func TestShellTool_Execute_EmptyCommand(t *testing.T) {
	ctx := context.Background()
	tool := ShellTool{}

	_, err := tool.Execute(ctx, map[string]any{})
	if err == nil {
		t.Error("expected error for empty command")
	}
}

func TestRegistry_Execute(t *testing.T) {
	ctx := context.Background()
	r := NewRegistry()

	out, err := r.Execute(ctx, "shell", map[string]any{"command": "echo ok"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out != "ok" {
		t.Errorf("got %q, want ok", out)
	}
}

func TestRegistry_Execute_ExecAlias(t *testing.T) {
	ctx := context.Background()
	r := NewRegistry()

	out, err := r.Execute(ctx, "exec", map[string]any{"command": "echo alias"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out != "alias" {
		t.Errorf("got %q, want alias", out)
	}
}
