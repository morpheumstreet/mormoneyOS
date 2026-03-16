package prompts

import (
	"bytes"
	"fmt"
)

// Version identifies a prompt template set.
type Version string

const (
	V1 Version = "v1"
)

// SystemPromptData holds data for rendering the system template.
// Populated by the agent loop from LoopConfig and runtime state.
type SystemPromptData struct {
	State          string
	Credits        string
	Tier           string
	TurnCount      int64
	Model          string
	LineageSummary string
	SkillsBlock    string
	GenesisPrompt  string
}

// BuildSystemPrompt renders the system prompt from the template for the given version.
// Returns the rendered string or an error if the template fails.
func BuildSystemPrompt(version Version, data SystemPromptData) (string, error) {
	tmpls, err := loadV1Templates()
	if err != nil {
		return "", fmt.Errorf("load templates: %w", err)
	}
	if version != V1 {
		return "", fmt.Errorf("unsupported prompt version: %s", version)
	}
	var buf bytes.Buffer
	if err := tmpls.ExecuteTemplate(&buf, "v1", data); err != nil {
		return "", fmt.Errorf("render system template: %w", err)
	}
	return buf.String(), nil
}

// GetCoTFooter returns the Chain-of-Thought instruction footer to append to the last user message.
// Kept small to avoid token bloat.
func GetCoTFooter() string {
	return `

Respond in this format:
Thought: 
Risk: 
Plan: 
Action: `
}

// CritiquePromptData holds turn content for the critique template.
type CritiquePromptData struct {
	Input     string
	Thinking  string
	ToolCalls string
}

// BuildCritiquePrompt renders the critique prompt from the template.
func BuildCritiquePrompt(data CritiquePromptData) (string, error) {
	tmpls, err := loadV1Templates()
	if err != nil {
		return "", fmt.Errorf("load templates: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpls.ExecuteTemplate(&buf, "critique", data); err != nil {
		return "", fmt.Errorf("render critique template: %w", err)
	}
	return buf.String(), nil
}
