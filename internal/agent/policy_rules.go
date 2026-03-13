package agent

import (
	"regexp"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// PolicyDecisionStore persists and queries policy decisions for audit and rate limits (TS-aligned).
type PolicyDecisionStore interface {
	InsertPolicyDecision(id, turnID, toolName, argsHash, riskLevel, decision, reason, source string) error
	CountRecentPolicyDecisions(toolName string, windowMs int64) (int, error)
}

// ValidationRule validates input formats.
type ValidationRule struct{}

func (ValidationRule) Priority() int { return 100 }

func (ValidationRule) Evaluate(tool types.ToolCall, _ string, _ types.RiskLevel) (allow bool, reason string) {
	// Basic validation: tool name must be non-empty
	if strings.TrimSpace(tool.Name) == "" {
		return false, "tool name is empty"
	}
	return true, ""
}

// PathProtectionRule blocks protected file access.
type PathProtectionRule struct {
	ProtectedWrites []string
	ProtectedReads  []string
}

func (PathProtectionRule) Priority() int { return 50 }

func (r PathProtectionRule) Evaluate(tool types.ToolCall, _ string, risk types.RiskLevel) (allow bool, reason string) {
	path, _ := tool.Args["path"].(string)
	if path == "" {
		path, _ = tool.Args["file_path"].(string)
	}
	if path == "" {
		return true, ""
	}
	for _, p := range r.ProtectedWrites {
		if strings.Contains(path, p) {
			return false, "path is protected: " + p
		}
	}
	for _, p := range r.ProtectedReads {
		if strings.Contains(path, p) {
			return false, "path is protected for read: " + p
		}
	}
	return true, ""
}

// AuthorityRule blocks dangerous tools from external sources.
type AuthorityRule struct{}

func (AuthorityRule) Priority() int { return 10 }

func (AuthorityRule) Evaluate(tool types.ToolCall, source string, risk types.RiskLevel) (allow bool, reason string) {
	if source == "creator" || source == "self" {
		return true, ""
	}
	if risk == types.RiskDangerous || risk == types.RiskForbidden {
		return false, "external/peer cannot invoke dangerous or forbidden tools"
	}
	return true, ""
}

// FinancialRule enforces TreasuryPolicy limits on transfer/send tools.
type FinancialRule struct {
	Policy *types.TreasuryPolicy
}

func (FinancialRule) Priority() int { return 80 }

func (r FinancialRule) Evaluate(tool types.ToolCall, _ string, _ types.RiskLevel) (allow bool, reason string) {
	policy := r.Policy
	if policy == nil {
		dp := types.DefaultTreasuryPolicy()
		policy = &dp
	}
	// Check tools that move value
	transferTools := map[string]bool{
		"transfer": true, "send_usdc": true, "send": true, "withdraw": true,
		"x402_pay": true, "topup": true, "topup_credits": true, "transfer_credits": true,
	}
	if !transferTools[strings.ToLower(tool.Name)] {
		return true, ""
	}
	amount := 0
	switch v := tool.Args["amount_cents"].(type) {
	case float64:
		amount = int(v)
	case int:
		amount = v
	}
	if amount <= 0 {
		return true, ""
	}
	if amount > policy.MaxSingleTransferCents {
		return false, "amount exceeds max single transfer"
	}
	if policy.MinReserveCents > 0 && amount > 0 {
		// Reserve check would need current balance from DB; skip for now
	}
	return true, ""
}

// CommandSafetyRule blocks dangerous shell/exec patterns.
type CommandSafetyRule struct {
	DangerousPatterns []*regexp.Regexp
}

func (CommandSafetyRule) Priority() int { return 60 }

func (r CommandSafetyRule) Evaluate(tool types.ToolCall, _ string, _ types.RiskLevel) (allow bool, reason string) {
	if tool.Name != "exec" && tool.Name != "shell" {
		return true, ""
	}
	cmd, _ := tool.Args["command"].(string)
	if cmd == "" {
		cmd, _ = tool.Args["cmd"].(string)
	}
	cmd = strings.TrimSpace(strings.ToLower(cmd))
	if cmd == "" {
		return true, ""
	}
	patterns := r.DangerousPatterns
	if len(patterns) == 0 {
		patterns = defaultDangerousPatterns
	}
	for _, re := range patterns {
		if re != nil && re.MatchString(cmd) {
			return false, "command matches blocked pattern"
		}
	}
	return true, ""
}

var defaultDangerousPatterns = func() []*regexp.Regexp {
	pat := []string{
		`rm\s+-rf\s+/`, `>\s*/dev/sd`, `mkfs\.`, `dd\s+if=`,
		`eval\s*\(`, `chmod\s+-R\s+777`,
	}
	out := make([]*regexp.Regexp, len(pat))
	for i, p := range pat {
		out[i], _ = regexp.Compile(p)
	}
	return out
}()

// CreateDefaultRules returns the 6-category rule set per design.
func CreateDefaultRules() []PolicyRule {
	return CreateDefaultRulesWithTreasury(nil, nil)
}

// RateLimitRule enforces per-tool/per-source rate limits (e.g. max exec/hour).
// When Checker is nil, always allows (no rate limiting).
type RateLimitRule struct {
	Checker func(toolName, source string) (exceeded bool)
}

func (RateLimitRule) Priority() int { return 70 }

func (r RateLimitRule) Evaluate(tool types.ToolCall, source string, _ types.RiskLevel) (allow bool, reason string) {
	if r.Checker == nil {
		return true, ""
	}
	if r.Checker(tool.Name, source) {
		return false, "rate limit exceeded for tool"
	}
	return true, ""
}

// rateLimitLimits defines TS-aligned limits: tool -> {windowMs, maxCount}.
var rateLimitLimits = map[string]struct {
	windowMs int64
	max      int
}{
	"update_genesis_prompt": {24 * 60 * 60 * 1000, 1}, // 1/day
	"edit_own_file":         {60 * 60 * 1000, 10},      // 10/hour
	"spawn_child":           {24 * 60 * 60 * 1000, 3},  // 3/day
}

// NewRateLimitChecker returns a Checker that queries policy_decisions (TS rate-limits aligned).
func NewRateLimitChecker(store PolicyDecisionStore) func(toolName, source string) bool {
	if store == nil {
		return nil
	}
	return func(toolName, source string) bool {
		lim, ok := rateLimitLimits[toolName]
		if !ok {
			return false
		}
		n, err := store.CountRecentPolicyDecisions(toolName, lim.windowMs)
		if err != nil {
			return false
		}
		return n >= lim.max
	}
}

// CreateDefaultRulesWithTreasury returns rules with optional TreasuryPolicy and optional PolicyDecisionStore for rate limits.
func CreateDefaultRulesWithTreasury(treasury *types.TreasuryPolicy, policyStore PolicyDecisionStore) []PolicyRule {
	rateRule := RateLimitRule{}
	if policyStore != nil {
		rateRule.Checker = NewRateLimitChecker(policyStore)
	}
	return []PolicyRule{
		ValidationRule{},
		PathProtectionRule{
			ProtectedWrites: []string{"constitution", "wallet", "state.db", "automaton.json", "SOUL.md"},
			ProtectedReads:  []string{"private", "api-key", ".env"},
		},
		FinancialRule{Policy: treasury},
		CommandSafetyRule{},
		rateRule,
		AuthorityRule{},
	}
}
