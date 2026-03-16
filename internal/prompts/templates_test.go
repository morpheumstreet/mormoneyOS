package prompts

import (
	"strings"
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

func TestBuildSystemPrompt_V1(t *testing.T) {
	data := SystemPromptData{
		State:          "running",
		Credits:        "12.50",
		Tier:           "normal",
		TurnCount:      42,
		Model:          "llama3.1-8B",
		LineageSummary: "",
		SkillsBlock:    "",
		GenesisPrompt:  "Be helpful.",
	}
	out, err := BuildSystemPrompt(V1, data)
	if err != nil {
		t.Fatalf("BuildSystemPrompt: %v", err)
	}
	if !strings.Contains(out, "automaton") {
		t.Error("expected 'automaton' in output")
	}
	if !strings.Contains(out, "running") {
		t.Error("expected 'running' in output")
	}
	if !strings.Contains(out, "12.50") {
		t.Error("expected credits in output")
	}
	if !strings.Contains(out, "Thought:") {
		t.Error("expected CoT instructions in output")
	}
	if !strings.Contains(out, "Be helpful.") {
		t.Error("expected genesis in output")
	}
}

func TestBuildSystemPrompt_UnsupportedVersion(t *testing.T) {
	_, err := BuildSystemPrompt("v99", SystemPromptData{})
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected 'unsupported' in error, got %v", err)
	}
}

func TestGetCoTFooter(t *testing.T) {
	footer := GetCoTFooter()
	if !strings.Contains(footer, "Thought:") {
		t.Error("expected Thought in footer")
	}
	if !strings.Contains(footer, "Risk:") {
		t.Error("expected Risk in footer")
	}
	if !strings.Contains(footer, "Plan:") {
		t.Error("expected Plan in footer")
	}
	if !strings.Contains(footer, "Action:") {
		t.Error("expected Action in footer")
	}
}

func TestRenderReactCoT(t *testing.T) {
	out, err := RenderReactCoT("mem block", "hist", "input")
	if err != nil {
		t.Fatalf("RenderReactCoT: %v", err)
	}
	if !strings.Contains(out, "mem block") {
		t.Error("expected memory block in output")
	}
	if !strings.Contains(out, "hist") {
		t.Error("expected history in output")
	}
	if !strings.Contains(out, "input") {
		t.Error("expected current input in output")
	}
	if !strings.Contains(out, "Think step-by-step") {
		t.Error("expected instructions in output")
	}
}

func TestFormatHistoryForReAct(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		out := FormatHistoryForReAct(nil)
		if out != "(no prior turns)" {
			t.Errorf("got %q", out)
		}
	})
	t.Run("with turns", func(t *testing.T) {
		turns := []state.Turn{
			{Input: "hi", Thinking: "hello"},
			{Input: "bye", Thinking: "goodbye"},
		}
		out := FormatHistoryForReAct(turns)
		if !strings.Contains(out, "hi") || !strings.Contains(out, "hello") {
			t.Errorf("expected hi/hello in output, got %q", out)
		}
		if !strings.Contains(out, "bye") || !strings.Contains(out, "goodbye") {
			t.Errorf("expected bye/goodbye in output, got %q", out)
		}
	})
}
