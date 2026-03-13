package agent

import (
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

func TestToolArgsHash_Deterministic(t *testing.T) {
	args := map[string]any{"a": 1, "b": "x"}
	h1 := ToolArgsHash(args)
	h2 := ToolArgsHash(args)
	if h1 != h2 {
		t.Errorf("ToolArgsHash not deterministic: %q != %q", h1, h2)
	}
}

func TestToolArgsHash_DifferentArgs(t *testing.T) {
	h1 := ToolArgsHash(map[string]any{"a": 1})
	h2 := ToolArgsHash(map[string]any{"a": 2})
	if h1 == h2 {
		t.Errorf("ToolArgsHash same for different args: %q", h1)
	}
}

func TestPolicyEngine_Evaluate_AllAllow(t *testing.T) {
	engine := NewPolicyEngine(CreateDefaultRules())
	tool := types.ToolCall{Name: "exec", Args: map[string]any{"cmd": "ls"}}
	allow, reason := engine.Evaluate(tool, "self", types.RiskSafe)
	if !allow {
		t.Errorf("Evaluate() allow = false, reason = %q", reason)
	}
}

func TestPolicyEngine_Evaluate_FirstDenyWins(t *testing.T) {
	engine := NewPolicyEngine(CreateDefaultRules())
	tool := types.ToolCall{Name: "", Args: map[string]any{}}
	allow, reason := engine.Evaluate(tool, "self", types.RiskSafe)
	if allow {
		t.Error("Evaluate() allow = true for empty tool name")
	}
	if reason != "tool name is empty" {
		t.Errorf("Evaluate() reason = %q, want 'tool name is empty'", reason)
	}
}

func TestCreateDefaultRules(t *testing.T) {
	rules := CreateDefaultRules()
	if len(rules) != 6 {
		t.Errorf("CreateDefaultRules() len = %d, want 6 (validation, path, financial, command-safety, rate-limit, authority)", len(rules))
	}
}

func TestFinancialRule_OverLimit(t *testing.T) {
	tp := types.DefaultTreasuryPolicy()
	tp.MaxSingleTransferCents = 100
	r := FinancialRule{Policy: &tp}
	tool := types.ToolCall{Name: "transfer", Args: map[string]any{"amount_cents": 150}}
	allow, _ := r.Evaluate(tool, "self", types.RiskSafe)
	if allow {
		t.Error("FinancialRule should deny amount over max single transfer")
	}
}

func TestFinancialRule_UnderLimit(t *testing.T) {
	tp := types.DefaultTreasuryPolicy()
	tp.MaxSingleTransferCents = 100
	r := FinancialRule{Policy: &tp}
	tool := types.ToolCall{Name: "transfer", Args: map[string]any{"amount_cents": 50}}
	allow, _ := r.Evaluate(tool, "self", types.RiskSafe)
	if !allow {
		t.Error("FinancialRule should allow amount under limit")
	}
}

func TestCommandSafetyRule_Dangerous(t *testing.T) {
	r := CommandSafetyRule{}
	tool := types.ToolCall{Name: "exec", Args: map[string]any{"command": "rm -rf /"}}
	allow, _ := r.Evaluate(tool, "self", types.RiskSafe)
	if allow {
		t.Error("CommandSafetyRule should deny rm -rf /")
	}
}

func TestCommandSafetyRule_Safe(t *testing.T) {
	r := CommandSafetyRule{}
	tool := types.ToolCall{Name: "exec", Args: map[string]any{"command": "ls -la"}}
	allow, _ := r.Evaluate(tool, "self", types.RiskSafe)
	if !allow {
		t.Error("CommandSafetyRule should allow safe commands")
	}
}
