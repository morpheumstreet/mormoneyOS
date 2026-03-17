package simulation

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/morpheumlabs/mormoneyos-go/internal/agent"
	"github.com/morpheumlabs/mormoneyos-go/internal/inference"
	"github.com/morpheumlabs/mormoneyos-go/internal/memory"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
	"github.com/morpheumlabs/mormoneyos-go/internal/tools"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

func TestSimulation_Run_NoCrashes(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sim.db")
	db, err := state.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	infClient := inference.NewStubClient("sim-stub")
	policy := agent.NewPolicyEngine(agent.CreateDefaultRulesWithTreasury(nil, db))
	reg := tools.NewRegistryWithOptions(&tools.RegistryOptions{
		Store:  db,
		Name:   "sim-test",
		Config: &types.AutomatonConfig{},
	})

	loop := agent.NewLoopWithOptions(agent.LoopOptions{
		Policy:          policy,
		Store:           db,
		Inference:       infClient,
		Tools:           reg,
		LineageStore:    db,
		MemoryRetriever: memory.NewTieredMemoryRetriever(db, memory.DefaultTierConfig()),
		MemoryIngester:  nil,
		Config: &agent.LoopConfig{
			Name:           "sim-test",
			InferenceModel:  "sim-stub",
			TokenLimits:    nil, // uses DefaultTokenLimits() when nil
		},
	})

	cfg := DefaultSimulationConfig()
	cfg.Days = 1
	cfg.ChaosLevel = ChaosNone
	cfg.Seed = 42

	startDate := time.Now().UTC().Truncate(24 * time.Hour).AddDate(0, 0, -1)
	replay := NewConstantTickReplay(startDate, time.Hour, 24, 50000.0)

	sim := NewSimulator(SimulatorOptions{
		Config: cfg,
		DB:     db,
		Loop:   loop,
		Replay: replay,
		Chaos:  NewChaosInjector(cfg.ChaosLevel, cfg.Seed),
	})

	ctx := context.Background()
	result, err := sim.Run(ctx)
	if err != nil {
		t.Fatalf("sim run: %v", err)
	}

	if result.TotalTurns == 0 {
		t.Error("expected at least one turn")
	}

	// Assert low crash rate (stub should never crash)
	crashes := 0
	for _, tr := range result.Turns {
		if tr.State == "" {
			crashes++
		}
	}
	if crashes > 0 {
		t.Errorf("unexpected crashes: %d", crashes)
	}
}

func TestChaosInjector_Levels(t *testing.T) {
	for _, level := range []ChaosLevel{ChaosNone, ChaosLow, ChaosMedium, ChaosHigh} {
		c := NewChaosInjector(level, 42)
		_ = c.ShouldInjectAPITimeout()
		_ = c.ShouldInjectEmptyLLMResponse()
		_ = c.ShouldInjectPriceFlashCrash()
	}
}

func TestReporter_Generate(t *testing.T) {
	dir := t.TempDir()
	r := NewReporter(dir, "json")
	result := &RunResult{
		StartTime:  time.Now().Add(-24 * time.Hour),
		EndTime:    time.Now(),
		TotalTurns: 10,
		Turns:      []TurnRecord{{Day: 0, Tick: 0, Time: time.Now(), State: "running"}},
	}
	if err := r.Generate(result); err != nil {
		t.Fatalf("generate: %v", err)
	}
	path := filepath.Join(dir, "sim-report.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("report file not created: %s", path)
	}
}
