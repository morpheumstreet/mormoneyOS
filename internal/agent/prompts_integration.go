package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/morpheumlabs/mormoneyos-go/internal/inference"
	"github.com/morpheumlabs/mormoneyos-go/internal/memory"
	"github.com/morpheumlabs/mormoneyos-go/internal/prompts"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

// BuildMessagesFromPrompts builds the message array using versioned templates and CoT forcing.
// Uses prompts.BuildSystemPrompt for system content, prompts.GetCoTFooter for the last user message,
// and reuses MessageTrimmer/BuildMessagesSafe for memory retrieval and token cap enforcement.
func BuildMessagesFromPrompts(
	ctx context.Context,
	version prompts.Version,
	systemData prompts.SystemPromptData,
	recentTurns []state.Turn,
	pendingInput string,
	memoryRetriever memory.MemoryRetriever,
	toolDefs []inference.ToolDefinition,
	limits TokenLimits,
	effectiveCap int,
	tok Tokenizer,
	log *slog.Logger,
) ([]inference.ChatMessage, error) {
	if tok == nil {
		tok = DefaultTokenizer
	}
	if limits.MaxInputTokens <= 0 {
		limits = DefaultTokenLimits()
	}

	// 1. Render system prompt from template
	systemPrompt, err := prompts.BuildSystemPrompt(version, systemData)
	if err != nil {
		return nil, fmt.Errorf("build system prompt: %w", err)
	}

	// 2. Append CoT footer to pending input
	pendingInputWithCoT := pendingInput + prompts.GetCoTFooter()

	// 3. Use MessageTrimmer when memory retriever is set, else BuildMessagesSafe
	var messages []inference.ChatMessage
	if memoryRetriever != nil {
		trimmer := NewMessageTrimmer(tok)
		messages, _ = trimmer.Trim(ctx, systemPrompt, recentTurns, pendingInputWithCoT, memoryRetriever, toolDefs, limits, effectiveCap, log)
	} else {
		messages = BuildMessagesSafe(systemPrompt, recentTurns, pendingInputWithCoT, "", toolDefs, limits, effectiveCap, tok, log)
	}

	return messages, nil
}
