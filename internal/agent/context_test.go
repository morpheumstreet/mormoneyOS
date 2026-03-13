package agent

import (
	"strings"
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

func TestBuildContextMessages_IncludesToolResults(t *testing.T) {
	turns := []state.Turn{
		{
			Input:     "list files",
			Thinking:  "I'll run ls",
			ToolCalls: `[{"name":"shell","result":"file1\nfile2","error":""}]`,
		},
	}
	msgs := BuildContextMessages("system", turns, "")

	// Should have system, user, assistant
	if len(msgs) < 3 {
		t.Fatalf("expected at least 3 messages, got %d", len(msgs))
	}
	asst := msgs[2]
	if asst.Role != "assistant" {
		t.Errorf("expected assistant role, got %s", asst.Role)
	}
	if !strings.Contains(asst.Content, "Tool results") {
		t.Errorf("assistant content should include tool results: %s", asst.Content)
	}
	if !strings.Contains(asst.Content, "file1") {
		t.Errorf("assistant content should include tool output: %s", asst.Content)
	}
}

func TestAppendToolResults_Empty(t *testing.T) {
	got := appendToolResults("think", "[]")
	if got != "think" {
		t.Errorf("empty tool calls should return thinking unchanged: %q", got)
	}
}

func TestAppendToolResults_WithResults(t *testing.T) {
	got := appendToolResults("I'll check", `[{"name":"shell","result":"ok","error":""}]`)
	if !strings.Contains(got, "shell") || !strings.Contains(got, "ok") {
		t.Errorf("should include tool name and result: %q", got)
	}
}
