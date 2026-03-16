package agent

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/inference"
	"github.com/morpheumlabs/mormoneyos-go/internal/memory"
	"github.com/morpheumlabs/mormoneyos-go/internal/prompts"
)

// ReflectionIngester ingests critique output into memory (optional extension of MemoryIngester).
type ReflectionIngester interface {
	IngestReflection(ctx context.Context, reflection *memory.ReflectionData) error
}

// Reflection holds the parsed critique output.
type Reflection struct {
	SuccessScore          float64
	Lessons               []string
	MemoryRecommendations []string
}

// ReflectionEngine runs self-critique on turns with impact. Uses cheap/fast model.
type ReflectionEngine struct {
	router *inference.ModelRouter
	log    *slog.Logger
}

// NewReflectionEngine creates an engine. Router provides the fast client for critique.
func NewReflectionEngine(router *inference.ModelRouter, log *slog.Logger) *ReflectionEngine {
	if log == nil {
		log = slog.Default()
	}
	return &ReflectionEngine{router: router, log: log}
}

// CritiqueTurnData holds the turn content for critique.
type CritiqueTurnData struct {
	TurnID     string
	Input      string
	Thinking   string
	ToolCalls  string
}

// CritiqueTurn runs critique on the turn and returns parsed reflection. Non-blocking on errors.
func (re *ReflectionEngine) CritiqueTurn(ctx context.Context, turn *CritiqueTurnData) (*Reflection, error) {
	if re.router == nil || turn == nil {
		return nil, nil
	}
	client, err := re.router.ClientForReflection(ctx)
	if err != nil || client == nil {
		return nil, err
	}
	prompt, err := prompts.BuildCritiquePrompt(prompts.CritiquePromptData{
		Input:     truncateForCritique(turn.Input, 500),
		Thinking:  truncateForCritique(turn.Thinking, 800),
		ToolCalls: truncateForCritique(turn.ToolCalls, 600),
	})
	if err != nil {
		re.log.Warn("critique prompt build failed", "err", err)
		return nil, err
	}
	msgs := []inference.ChatMessage{
		{Role: "user", Content: prompt},
	}
	opts := &inference.InferenceOptions{
		Model:       client.GetDefaultModel(),
		MaxTokens:   512,
		Temperature: 0.1,
	}
	resp, err := client.Chat(ctx, msgs, opts)
	if err != nil {
		re.log.Warn("critique inference failed", "err", err)
		return nil, err
	}
	RecordCritique()
	ref := parseCritiqueResponse(resp.Content)
	if ref != nil {
		RecordCritiqueSuccessScore(ref.SuccessScore)
	}
	if ref == nil {
		return nil, nil
	}
	return ref, nil
}

func truncateForCritique(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func parseCritiqueResponse(content string) *Reflection {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	var raw struct {
		SuccessScore          float64  `json:"success_score"`
		Lessons               []string `json:"lessons"`
		MemoryRecommendations []string `json:"memory_recommendations"`
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil
	}
	if raw.SuccessScore < 0 || raw.SuccessScore > 1 {
		raw.SuccessScore = 0.5
	}
	return &Reflection{
		SuccessScore:          raw.SuccessScore,
		Lessons:               raw.Lessons,
		MemoryRecommendations: raw.MemoryRecommendations,
	}
}
