package cmd

import (
	"context"
	"encoding/json"
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
	"github.com/morpheumlabs/mormoneyos-go/internal/identity"
	"github.com/morpheumlabs/mormoneyos-go/internal/inference"
	"github.com/morpheumlabs/mormoneyos-go/internal/memory"
	"github.com/morpheumlabs/mormoneyos-go/internal/replication"
	"github.com/morpheumlabs/mormoneyos-go/internal/social"
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

	// 2b. Identity bootstrap: multi-chain addresses (defaultChain + chainProviders), live wallet first
	primaryAddr, _ := identity.BootstrapIdentity(db, cfg)
	identity.EnsureCreatedAt(db)
	if primaryAddr == "" {
		primaryAddr = cfg.WalletAddress
	}

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

	// 5c. Social channels (Conway, Telegram, Discord)
	channels := social.NewChannelsFromConfig(cfg)

	// 5d. Bootstrap topup: buy minimum $5 credits from USDC when balance is low (TS-aligned)
	if conwayClient != nil && primaryAddr != "" {
		bootstrapTopupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		func() {
			defer cancel()
			creditsCents, _ := conwayClient.GetCreditsBalance(bootstrapTopupCtx)
			account, _, err := identity.GetWallet()
			if err != nil || account == nil {
				slog.Debug("bootstrap topup skipped: no wallet")
				return
			}
			result, err := conway.BootstrapTopup(bootstrapTopupCtx, conway.BootstrapTopupParams{
				APIURL:               cfg.ConwayAPIURL,
				Account:              account.PrivateKey(),
				Address:              account.Address(),
				CreditsCents:         creditsCents,
				CreditThresholdCents: 500,
				DefaultChain:         cfg.DefaultChain,
			})
			if err != nil {
				slog.Warn("bootstrap topup failed", "err", err)
				return
			}
			if result != nil && result.Success {
				slog.Info("bootstrap topup: credits added", "amount_usd", result.AmountUSD)
			} else if result != nil && result.Error != "" {
				slog.Warn("bootstrap topup skipped", "reason", result.Error)
			}
		}()
	}

	// 6. Agent loop (full ReAct when inference+store configured)
	reg := tools.NewRegistryWithOptions(&tools.RegistryOptions{
		Store:          db,
		Conway:         conwayClient,
		Name:           cfg.Name,
		ParentAddress:  primaryAddr,
		GenesisPrompt:  cfg.GenesisPrompt,
		Config:         cfg,
		ConfigTools:    cfg.Tools,
		InstalledDB:    db,
		PluginPaths:    cfg.PluginPaths,
		Channels:       channels,
		TunnelManager:  tunnelMgr,
		TunnelRegistry: tunnelReg,
	})
	loop := agent.NewLoopWithOptions(agent.LoopOptions{
		Policy:          policy,
		Store:           db,
		Inference:       infClient,
		Tools:           reg,
		LineageStore:    db,
		MemoryRetriever: memory.NewDBMemoryRetriever(db, memory.NewBudgetAllocator(memory.DefaultTokenBudget)),
		DisabledToolsGetter: func() []string {
			raw, ok, _ := db.GetKV("disabled_tools")
			if !ok || raw == "" {
				return nil
			}
			var list []string
			_ = json.Unmarshal([]byte(raw), &list)
			return list
		},
		Config: &agent.LoopConfig{
			Name:             cfg.Name,
			GenesisPrompt:    cfg.GenesisPrompt,
			CreatorMsg:       cfg.CreatorAddress,
			InferenceModel:   cfg.InferenceModel,
			LowComputeModel:  cfg.LowComputeModel,
			WalletAddress:    primaryAddr,
			SkillsConfig:     cfg.Skills,
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
			Channels:     channels,
			Address:      primaryAddr,
			HealthMonitor: &replication.ChildHealthMonitor{
				Conway: conwayClient,
				Store: db,
			},
			SandboxCleanup: &replication.SandboxCleanup{
				Conway: conwayClient,
				Store:  db,
			},
			Log: slog.Default(),
		})
	} else {
		daemon = heartbeat.NewDaemonWithOptions(heartbeat.DaemonOptions{
			TickInterval: tickInterval,
			WakeCheck:    wakeCheck,
			Tasks:        heartbeat.DefaultTasks(),
			Store:        db,
			WakeInserter: db,
			Config:       cfg,
			Channels:     channels,
			Address:      primaryAddr,
			Log:          slog.Default(),
		})
	}

	// 8. Web dashboard (moneyclaw-py aligned)
	webState := &web.RuntimeState{Running: true, AgentState: string(types.AgentStateWaking)}
	noWeb, _ := cmd.Flags().GetBool("no-web")
	webAddr, _ := cmd.Flags().GetString("web-addr")
	var creditsGetter web.CreditsGetter
	if conwayClient != nil {
		creditsGetter = conwayClient
	}
	var webSrv *web.Server
	if !noWeb {
		webSrv = web.NewServer(webAddr, webState, db, &web.ServerConfig{
			Name:            cfg.Name,
			WalletAddress:   primaryAddr,
			DefaultChain:    cfg.DefaultChain,
			Version:         web.Version,
			CreditsGetter:   creditsGetter,
			ChatClient:      infClient,
			ToolsLister:     reg,
			TunnelManager:   tunnelMgr,
			TunnelReloader:  func(tc *types.TunnelConfig) { tunnelMgr.Reload(tc) },
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

	// Shutdown web server and all background threads when backend closes
	defer func() {
		webState.UpdateState(false, "shutting_down", 0)
		if webSrv != nil {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer shutdownCancel()
			if err := webSrv.Shutdown(shutdownCtx); err != nil {
				slog.Warn("web server shutdown", "err", err)
			}
		}
	}()

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
			res := loop.RunOneTurn(ctx, agentState)
			if res.Err != nil {
				slog.Error("agent turn failed", "err", res.Err)
				continue
			}
			tickNum++
			if res.State == types.AgentStateSleeping {
				agentState = types.AgentStateSleeping
				idleTurns = 0
				// Ensure sleep_until is set (sleep tool sets it; finishReason/idle do not)
				if _, ok, _ := db.GetKV("sleep_until"); !ok {
					_ = db.SetKV("sleep_until", time.Now().Add(60*time.Second).UTC().Format(time.RFC3339))
				}
				slog.Info("agent sleeping")
			} else {
				agentState = res.State
				if res.WasIdle {
					idleTurns++
				} else {
					idleTurns = 0
				}
				if agentState == types.AgentStateRunning && loop.ShouldSleep(idleTurns) {
					agentState = types.AgentStateSleeping
					idleTurns = 0
					_ = db.SetKV("sleep_until", time.Now().Add(60*time.Second).UTC().Format(time.RFC3339))
					slog.Info("agent sleeping")
				}
			}

		case types.AgentStateSleeping:
			webState.UpdateState(true, string(agentState), tickNum)
			hasWake, err := db.HasUnconsumedWakeEvents()
			if err != nil {
				slog.Warn("check wake events", "err", err)
			}
			if hasWake {
				_, _ = db.ConsumeWakeEvents()
				_ = db.DeleteKV("sleep_until")
				agentState = types.AgentStateWaking
				slog.Info("wake event consumed, waking")
				continue
			}
			// Check sleep_until expiry (TS-aligned: wake when timer expires)
			if until, ok, _ := db.GetKV("sleep_until"); ok && until != "" {
				if t, err := time.Parse(time.RFC3339, until); err == nil && !t.After(time.Now()) {
					_ = db.DeleteKV("sleep_until")
					agentState = types.AgentStateWaking
					slog.Info("sleep_until expired, waking")
					continue
				}
			}
			// Sleep until next check; use min(wakeCheck, time until sleep_until) when set
			sleepDur := wakeCheck
			if until, ok, _ := db.GetKV("sleep_until"); ok && until != "" {
				if t, err := time.Parse(time.RFC3339, until); err == nil && t.After(time.Now()) {
					if d := time.Until(t); d < sleepDur && d > time.Second {
						sleepDur = d
					}
				}
			}
			time.Sleep(sleepDur)
		}
	}
}
