package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/prompts"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

func TestBuildMessagesFromPrompts_NoMemory(t *testing.T) {
	ctx := context.Background()
	systemData := prompts.SystemPromptData{
		State:         "running",
		Credits:       "10.00",
		Tier:          "normal",
		TurnCount:     5,
		Model:         "test",
		GenesisPrompt: "Be helpful.",
	}
	turns := []state.Turn{
		{Input: "hi", Thinking: "hello"},
	}
	limits := TokenLimits{MaxInputTokens: 8000, MaxHistoryTurns: 12}

	msgs, err := BuildMessagesFromPrompts(ctx, prompts.V1, systemData, turns, "what next?", nil, nil, limits, 0, &NaiveTokenizer{}, nil)
	if err != nil {
		t.Fatalf("BuildMessagesFromPrompts: %v", err)
	}
	if len(msgs) < 3 {
		t.Errorf("expected at least 3 messages (system, history, input), got %d", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Errorf("first message role = %q, want system", msgs[0].Role)
	}
	// Last user message should contain CoT footer
	lastContent := ""
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			lastContent = msgs[i].Content
			break
		}
	}
	if lastContent != "" && !strings.Contains(lastContent, "Thought:") && !strings.Contains(lastContent, "Action:") {
		preview := lastContent
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		t.Errorf("last user message should contain CoT instructions, got %q", preview)
	}
}

func TestBuildMessagesFromPrompts_UnsupportedVersion(t *testing.T) {
	ctx := context.Background()
	_, err := BuildMessagesFromPrompts(ctx, prompts.Version("v99"), prompts.SystemPromptData{}, nil, "hi", nil, nil, DefaultTokenLimits(), 0, nil, nil)
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
}
