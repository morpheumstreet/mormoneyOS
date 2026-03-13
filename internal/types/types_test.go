package types

import (
	"testing"
)

func TestDefaultTreasuryPolicy(t *testing.T) {
	tp := DefaultTreasuryPolicy()
	if tp.MaxSingleTransferCents != 5000 {
		t.Errorf("MaxSingleTransferCents = %d, want 5000", tp.MaxSingleTransferCents)
	}
	if tp.MinReserveCents != 100 {
		t.Errorf("MinReserveCents = %d, want 100", tp.MinReserveCents)
	}
	if len(tp.X402AllowedDomains) == 0 {
		t.Error("X402AllowedDomains empty")
	}
}

func TestAgentStateConstants(t *testing.T) {
	states := []AgentState{
		AgentStateSetup, AgentStateWaking, AgentStateRunning,
		AgentStateSleeping, AgentStateLowCompute, AgentStateCritical, AgentStateDead,
	}
	for _, s := range states {
		if s == "" {
			t.Error("AgentState constant is empty")
		}
	}
}

func TestSurvivalTierConstants(t *testing.T) {
	tiers := []SurvivalTier{
		SurvivalTierHigh, SurvivalTierNormal, SurvivalTierLowCompute,
		SurvivalTierCritical, SurvivalTierDead,
	}
	for _, tier := range tiers {
		if tier == "" {
			t.Error("SurvivalTier constant is empty")
		}
	}
}

func TestRiskLevelConstants(t *testing.T) {
	levels := []RiskLevel{RiskSafe, RiskCaution, RiskDangerous, RiskForbidden}
	for _, l := range levels {
		if l == "" {
			t.Error("RiskLevel constant is empty")
		}
	}
}
