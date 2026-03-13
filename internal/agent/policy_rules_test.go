package agent

import (
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

func TestValidationRule_EmptyName(t *testing.T) {
	r := ValidationRule{}
	tool := types.ToolCall{Name: "", Args: nil}
	allow, _ := r.Evaluate(tool, "self", types.RiskSafe)
	if allow {
		t.Error("ValidationRule empty name should deny")
	}
}

func TestValidationRule_WhitespaceName(t *testing.T) {
	r := ValidationRule{}
	tool := types.ToolCall{Name: "  ", Args: nil}
	allow, _ := r.Evaluate(tool, "self", types.RiskSafe)
	if allow {
		t.Error("ValidationRule whitespace name should deny")
	}
}

func TestValidationRule_ValidName(t *testing.T) {
	r := ValidationRule{}
	tool := types.ToolCall{Name: "exec", Args: nil}
	allow, _ := r.Evaluate(tool, "self", types.RiskSafe)
	if !allow {
		t.Error("ValidationRule valid name should allow")
	}
}

func TestPathProtectionRule_ProtectedWrite(t *testing.T) {
	r := PathProtectionRule{
		ProtectedWrites: []string{"constitution", "wallet"},
		ProtectedReads:  []string{},
	}
	tool := types.ToolCall{Name: "write", Args: map[string]any{"path": "/x/constitution"}}
	allow, reason := r.Evaluate(tool, "self", types.RiskSafe)
	if allow {
		t.Error("PathProtectionRule constitution write should deny")
	}
	if reason == "" {
		t.Error("PathProtectionRule should return reason")
	}
}

func TestPathProtectionRule_ProtectedRead(t *testing.T) {
	r := PathProtectionRule{
		ProtectedWrites: []string{},
		ProtectedReads:  []string{"api-key"},
	}
	tool := types.ToolCall{Name: "read", Args: map[string]any{"path": "/x/api-key"}}
	allow, _ := r.Evaluate(tool, "self", types.RiskSafe)
	if allow {
		t.Error("PathProtectionRule api-key read should deny")
	}
}

func TestPathProtectionRule_SafePath(t *testing.T) {
	r := PathProtectionRule{
		ProtectedWrites: []string{"constitution"},
		ProtectedReads:  []string{"api-key"},
	}
	tool := types.ToolCall{Name: "read", Args: map[string]any{"path": "/tmp/foo"}}
	allow, _ := r.Evaluate(tool, "self", types.RiskSafe)
	if !allow {
		t.Error("PathProtectionRule safe path should allow")
	}
}

func TestPathProtectionRule_NoPathArg(t *testing.T) {
	r := PathProtectionRule{
		ProtectedWrites: []string{"constitution"},
		ProtectedReads:  []string{},
	}
	tool := types.ToolCall{Name: "exec", Args: map[string]any{}}
	allow, _ := r.Evaluate(tool, "self", types.RiskSafe)
	if !allow {
		t.Error("PathProtectionRule no path should allow (skip)")
	}
}

func TestPathProtectionRule_FilePathArg(t *testing.T) {
	r := PathProtectionRule{
		ProtectedWrites: []string{"state.db"},
		ProtectedReads:  []string{},
	}
	tool := types.ToolCall{Name: "write", Args: map[string]any{"file_path": "/x/state.db"}}
	allow, _ := r.Evaluate(tool, "self", types.RiskSafe)
	if allow {
		t.Error("PathProtectionRule file_path with state.db should deny")
	}
}

func TestAuthorityRule_Creator(t *testing.T) {
	r := AuthorityRule{}
	tool := types.ToolCall{Name: "exec", Args: map[string]any{}}
	allow, _ := r.Evaluate(tool, "creator", types.RiskDangerous)
	if !allow {
		t.Error("AuthorityRule creator should allow dangerous")
	}
}

func TestAuthorityRule_Self(t *testing.T) {
	r := AuthorityRule{}
	tool := types.ToolCall{Name: "exec", Args: map[string]any{}}
	allow, _ := r.Evaluate(tool, "self", types.RiskDangerous)
	if !allow {
		t.Error("AuthorityRule self should allow dangerous")
	}
}

func TestAuthorityRule_ExternalDangerous(t *testing.T) {
	r := AuthorityRule{}
	tool := types.ToolCall{Name: "exec", Args: map[string]any{}}
	allow, reason := r.Evaluate(tool, "external", types.RiskDangerous)
	if allow {
		t.Error("AuthorityRule external+dangerous should deny")
	}
	if reason == "" {
		t.Error("AuthorityRule should return reason")
	}
}

func TestAuthorityRule_ExternalSafe(t *testing.T) {
	r := AuthorityRule{}
	tool := types.ToolCall{Name: "exec", Args: map[string]any{}}
	allow, _ := r.Evaluate(tool, "external", types.RiskSafe)
	if !allow {
		t.Error("AuthorityRule external+safe should allow")
	}
}
