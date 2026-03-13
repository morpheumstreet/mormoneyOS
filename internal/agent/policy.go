package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// PolicyEngine evaluates tool calls before execution. First deny wins.
type PolicyEngine struct {
	rules []PolicyRule
}

// PolicyRule evaluates a single tool call.
type PolicyRule interface {
	Evaluate(tool types.ToolCall, source string, risk types.RiskLevel) (allow bool, reason string)
	Priority() int
}

// NewPolicyEngine creates an engine with default rules.
func NewPolicyEngine(rules []PolicyRule) *PolicyEngine {
	return &PolicyEngine{rules: rules}
}

// Evaluate runs all rules; returns allow, reason. First deny wins.
func (e *PolicyEngine) Evaluate(tool types.ToolCall, source string, risk types.RiskLevel) (allow bool, reason string) {
	for _, r := range e.rules {
		allow, reason := r.Evaluate(tool, source, risk)
		if !allow {
			return false, reason
		}
	}
	return true, ""
}

// ToolArgsHash returns a hash of tool args for audit.
func ToolArgsHash(args map[string]any) string {
	b, _ := json.Marshal(args)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:8])
}
