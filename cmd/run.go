package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/morpheumlabs/mormoneyos-go/internal/agent"
	"github.com/morpheumlabs/mormoneyos-go/internal/config"
	"github.com/morpheumlabs/mormoneyos-go/internal/conway"
	"github.com/morpheumlabs/mormoneyos-go/internal/heartbeat"
	"github.com/morpheumlabs/mormoneyos-go/internal/inference"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
	"github.com/morpheumlabs/mormoneyos-go/internal/tools"
	"github.com/morpheumlabs/mormoneyos-go/internal/tunnel"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
	"github.com/morpheumlabs/mormoneyos-go/internal/web"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start runtime (agent loop + heartbeat)",
	Long:  `Bootstrap and run the automaton. Alternates between running and sleeping.`,
	RunE:  runRun,
}

func init() {
	runCmd.Flags().Duration("tick-interval", 60*time.Second, "Heartbeat tick interval")
	runCmd.Flags().Duration("wake-check", 30*time.Second, "Wake event check interval during sleep")
	runCmd.Flags().Bool("no-telegram", false, "Disable Telegram bot")
	runCmd.Flags().Bool("no-web", false, "Disable web dashboard")
	runCmd.Flags().String("web-addr", ":8080", "Web dashboard listen address")
}

func runRun(cmd *cobra.Command, args []string) error {
	// 1. Config load
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg == nil {
		return fmt.Errorf("no config: run 'moneyclaw setup' first")
	}

	// 2. Database init
	dbPath := config.ResolvePath(cfg.DBPath)
	db, err := state.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	// 2b. Identity bootstrap: persist address and default_chain to identity table
	_ = db.SetIdentity("address", cfg.WalletAddress)
	_ = db.SetIdentity("default_chain", cfg.DefaultChain)

	// 3. Policy engine (with treasury policy and DB-backed rate limits)
	policy := agent.NewPolicyEngine(agent.CreateDefaultRulesWithTreasury(cfg.TreasuryPolicy, db))

	// 4. Inference client (real when OpenAI/Conway keys set, else stub)
	infClient := inference.NewClientFromConfig(cfg)

	// 5. Conway client (when configured) — shared by agent, heartbeat, web
	var conwayClient conway.Client
	if cfg.ConwayAPIURL != "" && cfg.ConwayAPIKey != "" {
		conwayClient = conway.NewHTTPClient(cfg.ConwayAPIURL, cfg.ConwayAPIKey)
	}
	var creditsFn func(context.Context) int64
	if conwayClient != nil {
		creditsFn = func(ctx context.Context) int64 {
			c, _ := conwayClient.GetCreditsBalance(ctx)
			return c
		}
	}

	// 5b. Tunnel (expose_port, remove_port, tunnel_status)
	tunnelReg, tunnelMgr := tunnel.NewFromConfig(cfg.Tunnel)

	// 6. Agent loop (full ReAct when inference+store configured)
	reg := tools.NewRegistryWithOptions(&tools.RegistryOptions{
		Store:          db,
		Conway:         conwayClient,
		Name:           cfg.Name,
		ConfigTools:    cfg.Tools,
		InstalledDB:    db,
		PluginPaths:    cfg.PluginPaths,
		TunnelManager:  tunnelMgr,
		TunnelRegistry: tunnelReg,
	})
	loop := agent.NewLoopWithOptions(agent.LoopOptions{
		Policy:    policy,
		Store:     db,
		Inference: infClient,
		Tools:     reg,
		Config: &agent.LoopConfig{
			Name:           cfg.Name,
			GenesisPrompt:  cfg.GenesisPrompt,
			CreatorMsg:     cfg.CreatorAddress,
			InferenceModel: cfg.InferenceModel,
			WalletAddress:  cfg.WalletAddress,
		},
		CreditsFn: creditsFn,
		Log:       slog.Default(),
	})

	// 7. Heartbeat daemon (full task context when Conway+config available)
	tickInterval, _ := cmd.Flags().GetDuration("tick-interval")
	wakeCheck, _ := cmd.Flags().GetDuration("wake-check")
	var daemon *heartbeat.Daemon
	if conwayClient != nil {
		daemon = heartbeat.NewDaemonWithOptions(heartbeat.DaemonOptions{
			TickInterval: tickInterval,
			WakeCheck:    wakeCheck,
			Tasks:        heartbeat.DefaultTasks(),
			Store:        db,
			WakeInserter: db,
			CreditsFn:    creditsFn,
			Config:       cfg,
			Conway:       conwayClient,
			Address:      cfg.WalletAddress,
			Log:          slog.Default(),
		})
	} else {
		daemon = heartbeat.NewDaemonWithWakeInserter(tickInterval, wakeCheck, heartbeat.DefaultTasks(), db, slog.Default())
	}

	// 8. Web dashboard (moneyclaw-py aligned)
	webState := &web.RuntimeState{Running: true, AgentState: string(types.AgentStateWaking)}
	noWeb, _ := cmd.Flags().GetBool("no-web")
	webAddr, _ := cmd.Flags().GetString("web-addr")
	var creditsGetter web.CreditsGetter
	if conwayClient != nil {
		creditsGetter = conwayClient
	}
	if !noWeb {
		webSrv := web.NewServer(webAddr, webState, db, &web.ServerConfig{
			Name:          cfg.Name,
			WalletAddress: cfg.WalletAddress,
			Version:       web.Version,
			CreditsGetter: creditsGetter,
		}, slog.Default())
		go func() {
			if err := webSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("web server error", "err", err)
			}
		}()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("shutdown signal received")
		cancel()
	}()

	daemon.Start(ctx)
	defer daemon.Stop()

	// 9. Main loop: waking -> running -> sleeping -> waking
	agentState := types.AgentStateWaking
	idleTurns := 0
	tickNum := int64(0)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		switch agentState {
		case types.AgentStateWaking, types.AgentStateRunning:
			webState.UpdateState(true, string(agentState), tickNum)
			if webState.IsPaused() {
				time.Sleep(wakeCheck)
				continue
			}
			newState, err := loop.RunOneTurn(ctx, agentState)
			if err != nil {
				slog.Error("agent turn failed", "err", err)
				continue
			}
			tickNum++
			if newState == agentState && loop.ShouldSleep(idleTurns) {
				agentState = types.AgentStateSleeping
				idleTurns = 0
				slog.Info("agent sleeping")
			} else {
				agentState = newState
				idleTurns++
			}

		case types.AgentStateSleeping:
			webState.UpdateState(true, string(agentState), tickNum)
			hasWake, err := db.HasUnconsumedWakeEvents()
			if err != nil {
				slog.Warn("check wake events", "err", err)
			}
			if hasWake {
				_, _ = db.ConsumeWakeEvents()
				agentState = types.AgentStateWaking
				slog.Info("wake event consumed, waking")
				continue
			}
			time.Sleep(wakeCheck)
		}
	}
}
