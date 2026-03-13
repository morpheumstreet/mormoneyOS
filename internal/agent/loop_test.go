package agent

import (
	"context"
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

func TestRunOneTurn(t *testing.T) {
	policy := NewPolicyEngine(CreateDefaultRules())
	loop := NewLoop(policy, nil)
	ctx := context.Background()

	state, err := loop.RunOneTurn(ctx, types.AgentStateWaking)
	if err != nil {
		t.Fatalf("RunOneTurn() err = %v", err)
	}
	if state != types.AgentStateRunning {
		t.Errorf("RunOneTurn() state = %q, want running", state)
	}
}

func TestShouldSleep_IdleTurns2(t *testing.T) {
	loop := NewLoop(nil, nil)
	if loop.ShouldSleep(2) {
		t.Error("ShouldSleep(2) = true, want false")
	}
}

func TestShouldSleep_IdleTurns3(t *testing.T) {
	loop := NewLoop(nil, nil)
	if !loop.ShouldSleep(3) {
		t.Error("ShouldSleep(3) = false, want true")
	}
}
