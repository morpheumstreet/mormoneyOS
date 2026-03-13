package memory

import (
	"strings"
	"testing"
)

func TestFormatMemoryBlock_Empty(t *testing.T) {
	if got := FormatMemoryBlock(nil); got != "" {
		t.Errorf("FormatMemoryBlock(nil) = %q, want \"\"", got)
	}
	if got := FormatMemoryBlock(&MemoryBlock{}); got != "" {
		t.Errorf("FormatMemoryBlock(empty) = %q, want \"\"", got)
	}
}

func TestFormatMemoryBlock_Facts(t *testing.T) {
	block := &MemoryBlock{
		Facts: []string{"User prefers dark mode", "API key stored in env"},
	}
	got := FormatMemoryBlock(block)
	if !strings.Contains(got, "### Known Facts") {
		t.Error("missing Known Facts section")
	}
	if !strings.Contains(got, "User prefers dark mode") {
		t.Error("missing fact")
	}
	if !strings.HasPrefix(got, "## Memory") {
		t.Errorf("expected ## Memory prefix, got %q", got[:20])
	}
}

func TestFormatMemoryBlock_GoalsAndProcedures(t *testing.T) {
	block := &MemoryBlock{
		Goals: []string{"Complete migration"},
		Procedures: []ProcedureEntry{
			{Name: "deploy", Steps: 5},
		},
	}
	got := FormatMemoryBlock(block)
	if !strings.Contains(got, "### Active Goals") {
		t.Error("missing Active Goals section")
	}
	if !strings.Contains(got, "Complete migration") {
		t.Error("missing goal")
	}
	if !strings.Contains(got, "### Known Procedures") {
		t.Error("missing Known Procedures section")
	}
	if !strings.Contains(got, "deploy: 5 steps") {
		t.Error("missing procedure")
	}
}
