// Package dto holds shared request/response models and format helpers (DRY with REST/MCP).
package dto

import (
	"encoding/json"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/entity"
)

// FormatSkills returns skills as JSON string for MCP content format.
func FormatSkills(skills []entity.Skill) string {
	if len(skills) == 0 {
		return `{"skills":[]}`
	}
	b, _ := json.Marshal(map[string]any{"skills": skills})
	return string(b)
}

// FormatSkill returns a single skill as JSON string.
func FormatSkill(skill *entity.Skill) string {
	if skill == nil {
		return `{"skill":null}`
	}
	b, _ := json.Marshal(map[string]any{"skill": skill})
	return string(b)
}
