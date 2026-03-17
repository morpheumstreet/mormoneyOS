package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/morpheumlabs/mormoneyos-go/internal/agent"
	"github.com/morpheumlabs/mormoneyos-go/internal/config"
	"github.com/morpheumlabs/mormoneyos-go/internal/inference"
	"github.com/morpheumlabs/mormoneyos-go/internal/memory"
	"github.com/morpheumlabs/mormoneyos-go/internal/simulation"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
	"github.com/morpheumlabs/mormoneyos-go/internal/tools"
	"github.com/spf13/cobra"
)

var simCmd = &cobra.Command{
	Use:   "sim",
	Short: "Run simulation/backtest mode",
	Long:  `Run deterministic simulation to validate agent stability, memory pipeline, and token handling before real-money runs.`,
	RunE:  runSim,
}

func init() {
	simCmd.Flags().Int("days", 7, "Number of simulated days")
	simCmd.Flags().Int("speed", 100, "Speed multiplier (e.g. 100 = 100x realtime)")
	simCmd.Flags().String("chaos", "none", "Chaos level: none, low, medium, high")
	simCmd.Flags().Int64("seed", 42, "Random seed for reproducibility")
	simCmd.Flags().String("report", "json", "Report format: json, html")
	simCmd.Flags().String("config", "", "Path to simulation config (optional)")
	simCmd.Flags().String("output", "sim-results", "Output directory for reports")
}

func runSim(cmd *cobra.Command, args []string) error {
	days, _ := cmd.Flags().GetInt("days")
	speed, _ := cmd.Flags().GetInt("speed")
	chaosStr, _ := cmd.Flags().GetString("chaos")
	seed, _ := cmd.Flags().GetInt64("seed")
	reportFmt, _ := cmd.Flags().GetString("report")
	outputDir, _ := cmd.Flags().GetString("output")

	simCfg := simulation.DefaultSimulationConfig()
	simCfg.Days = days
	if simCfg.Days <= 0 {
		simCfg.Days = 7
	}
	simCfg.SpeedMultiplier = speed
	simCfg.Seed = seed
	simCfg.ReportFormat = reportFmt
	simCfg.ReportOutputDir = outputDir

	switch chaosStr {
	case "low":
		simCfg.ChaosLevel = simulation.ChaosLow
	case "medium":
		simCfg.ChaosLevel = simulation.ChaosMedium
	case "high":
		simCfg.ChaosLevel = simulation.ChaosHigh
	default:
		simCfg.ChaosLevel = simulation.ChaosNone
	}

	// Load main config for agent setup (optional)
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	// Use temp DB for simulation (isolated from production state)
	dbPath := filepath.Join(os.TempDir(), fmt.Sprintf("mormoneyos-sim-%d.db", time.Now().UnixNano()))
	db, err := state.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open sim db: %w", err)
	}
	defer db.Close()
	defer os.Remove(dbPath) // clean up temp file

	// Stub inference for deterministic sim (no real LLM calls)
	infClient := inference.NewStubClient("sim-stub")

	// Minimal tools registry (no Conway, no social — sim-safe)
	reg := tools.NewRegistryWithOptions(&tools.RegistryOptions{
		Store:  db,
		Name:   cfg.Name,
		Config: cfg,
	})

	// Policy engine
	policy := agent.NewPolicyEngine(agent.CreateDefaultRulesWithTreasury(cfg.TreasuryPolicy, db))

	// Memory service: disabled in sim (pass explicit nil interface to avoid nil pointer)
	// Full auto-ingestion pipeline can be enabled in future with stub inference for extraction.

	// Agent loop (same as production, with stub inference)
	loop := agent.NewLoopWithOptions(agent.LoopOptions{
		Policy:       policy,
		Store:        db,
		Inference:    infClient,
		Tools:        reg,
		LineageStore: db,
		MemoryRetriever: memory.NewTieredMemoryRetriever(db, memory.DefaultTierConfig()),
		MemoryIngester:  nil, // explicit nil interface; memSvc would make interface non-nil when nil
		Config: agent.BuildLoopConfig(cfg, &agent.BuildLoopConfigOpts{InferenceModel: "sim-stub"}),
		Log: slog.Default(),
	})

	// Replay provider (constant tick when no market data)
	startDate := time.Now().UTC().Truncate(24 * time.Hour).AddDate(0, 0, -simCfg.Days)
	replay := simulation.NewConstantTickReplay(startDate, time.Hour, 24, 50000.0)

	sim := simulation.NewSimulator(simulation.SimulatorOptions{
		Config: simCfg,
		DB:     db,
		Loop:   loop,
		MemSvc: nil,
		Replay: replay,
		Chaos:  simulation.NewChaosInjector(simCfg.ChaosLevel, simCfg.Seed),
		Log:    slog.Default(),
	})

	slog.Info("simulation starting", "days", simCfg.Days, "chaos", simCfg.ChaosLevel, "seed", simCfg.Seed)

	ctx := context.Background()
	result, err := sim.Run(ctx)
	if err != nil {
		return fmt.Errorf("sim run: %w", err)
	}

	reporter := simulation.NewReporter(simCfg.ReportOutputDir, simCfg.ReportFormat)
	if err := reporter.Generate(result); err != nil {
		return fmt.Errorf("generate report: %w", err)
	}

	slog.Info("simulation complete",
		"turns", result.TotalTurns,
		"report", filepath.Join(simCfg.ReportOutputDir, "sim-report."+simCfg.ReportFormat),
	)

	return nil
}
