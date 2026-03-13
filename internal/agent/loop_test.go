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

	res := loop.RunOneTurn(ctx, types.AgentStateWaking)
	if res.Err != nil {
		t.Fatalf("RunOneTurn() err = %v", res.Err)
	}
	if res.State != types.AgentStateRunning {
		t.Errorf("RunOneTurn() state = %q, want running", res.State)
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
