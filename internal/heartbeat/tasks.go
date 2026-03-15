package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/morpheumlabs/mormoneyos-go/internal/conway"
	"github.com/morpheumlabs/mormoneyos-go/internal/replication"
	"github.com/morpheumlabs/mormoneyos-go/internal/social"
	"github.com/morpheumlabs/mormoneyos-go/internal/soul"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// DefaultTasks returns built-in heartbeat tasks per design (TS BUILTIN_TASKS-aligned).
// 11 tasks: heartbeat_ping, check_credits, check_usdc_balance, check_social_inbox,
// check_for_updates, soul_reflection, refresh_models, check_child_health, prune_dead_children,
// health_check, report_metrics.
func DefaultTasks() []Task {
	return []Task{
		{Name: "heartbeat_ping", Schedule: "*/15 * * * *", Run: runHeartbeatPing},
		{Name: "check_credits", Schedule: "0 */6 * * *", Run: runCheckCredits},
		{Name: "check_usdc_balance", Schedule: "*/5 * * * *", Run: runCheckUSDCBalance},
		{Name: "check_social_inbox", Schedule: "*/10 * * * * *", Run: runCheckSocialInbox},
		{Name: "check_for_updates", Schedule: "0 */4 * * *", Run: runCheckForUpdates},
		{Name: "soul_reflection", Schedule: "0 */12 * * *", Run: runSoulReflection},
		{Name: "refresh_models", Schedule: "0 */6 * * *", Run: runRefreshModels},
		{Name: "check_child_health", Schedule: "*/30 * * * *", Run: runCheckChildHealth},
		{Name: "prune_dead_children", Schedule: "0 */6 * * *", Run: runPruneDeadChildren},
		{Name: "health_check", Schedule: "*/30 * * * *", Run: runHealthCheck},
		{Name: "report_metrics", Schedule: "0 * * * *", Run: runReportMetrics},
	}
}

func runHeartbeatPing(ctx context.Context, tc *TaskContext) (bool, string, error) {
	if tc == nil {
		return false, "", nil
	}
	credits := tc.Tick.CreditBalance
	tier := tc.Tick.SurvivalTier
	state, _, _ := tc.DB.GetAgentState()
	if state == "" {
		state = "running"
	}
	startTime, _, _ := tc.DB.GetKV("start_time")
	if startTime == "" {
		startTime = time.Now().UTC().Format(time.RFC3339)
		_ = tc.DB.SetKV("start_time", startTime)
	}
	start, _ := time.Parse(time.RFC3339, startTime)
	uptimeSec := int64(time.Since(start).Seconds())

	payload := map[string]any{
		"name":           tc.Config.Name,
		"address":        tc.Address,
		"state":          state,
		"creditsCents":   credits,
		"uptimeSeconds":  uptimeSec,
		"version":        "0.1.0",
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"tier":           string(tier),
	}
	b, _ := json.Marshal(payload)
	_ = tc.DB.SetKV("last_heartbeat_ping", string(b))

	if tier == types.SurvivalTierCritical || tier == types.SurvivalTierDead {
		distress := map[string]any{
			"level":         string(tier),
			"name":          tc.Config.Name,
			"address":       tc.Address,
			"creditsCents":  credits,
			"fundingHint":   "Use credit transfer API from a creator runtime to top this wallet up.",
			"timestamp":     time.Now().UTC().Format(time.RFC3339),
		}
		db, _ := json.Marshal(distress)
		_ = tc.DB.SetKV("last_distress", string(db))
		return true, fmt.Sprintf("Distress: %s. Credits: $%.2f. Need funding.", tier, float64(credits)/100), nil
	}
	return false, "", nil
}

const deadGracePeriod = 3_600_000 // 1 hour in ms

func runCheckCredits(ctx context.Context, tc *TaskContext) (bool, string, error) {
	if tc == nil {
		return false, "", nil
	}
	credits := tc.Tick.CreditBalance
	tier := tc.Tick.SurvivalTier
	now := time.Now().UTC().Format(time.RFC3339)

	check := map[string]any{"credits": credits, "tier": string(tier), "timestamp": now}
	b, _ := json.Marshal(check)
	_ = tc.DB.SetKV("last_credit_check", string(b))

	prevTier, _, _ := tc.DB.GetKV("prev_credit_tier")
	_ = tc.DB.SetKV("prev_credit_tier", string(tier))

	// Dead state: zero credits for >1 hour
	if tier == types.SurvivalTierCritical && credits == 0 {
		zeroSince, _, _ := tc.DB.GetKV("zero_credits_since")
		if zeroSince == "" {
			_ = tc.DB.SetKV("zero_credits_since", now)
		} else {
			t0, err := time.Parse(time.RFC3339, zeroSince)
			if err == nil {
				elapsed := time.Since(t0).Milliseconds()
				if elapsed >= deadGracePeriod {
					_ = tc.DB.SetAgentState("dead")
					return true, fmt.Sprintf("Dead: zero credits for %d minutes. Need funding.", elapsed/60000), nil
				}
			}
		}
	} else {
		_ = tc.DB.DeleteKV("zero_credits_since")
	}

	// Wake if tier dropped to critical
	if prevTier != "" && prevTier != string(tier) && tier == types.SurvivalTierCritical {
		return true, fmt.Sprintf("Credits dropped to %s tier: $%.2f", tier, float64(credits)/100), nil
	}
	return false, "", nil
}

func runCheckUSDCBalance(ctx context.Context, tc *TaskContext) (bool, string, error) {
	if tc == nil {
		return false, "", nil
	}
	address := tc.Address
	if address == "" {
		return false, "", nil
	}
	chains := []string{"eip155:8453"}
	if tc.Config != nil && tc.Config.DefaultChain != "" {
		chains = []string{tc.Config.DefaultChain}
	}
	var providers map[string]conway.USDCChainProvider
	if tc.Config != nil && len(tc.Config.ChainProviders) > 0 {
		providers = make(map[string]conway.USDCChainProvider)
		chains = make([]string, 0, len(tc.Config.ChainProviders))
		for chain, cfg := range tc.Config.ChainProviders {
			providers[chain] = conway.USDCChainProvider{RPCURL: cfg.RPCURL, USDCAddress: cfg.USDCAddress}
			chains = append(chains, chain)
		}
	}
	results, err := conway.GetUSDCBalanceMulti(ctx, address, chains, providers)
	if err != nil {
		_ = tc.DB.SetKV("last_usdc_check", `{"error":"`+err.Error()+`","timestamp":"`+time.Now().UTC().Format(time.RFC3339)+`"}`)
		return false, "", nil
	}
	var total float64
	byChain := make(map[string]float64)
	for _, r := range results {
		total += r.Balance
		byChain[r.Chain] = r.Balance
	}
	check := map[string]any{
		"balance": total, "byChain": byChain, "credits": tc.Tick.CreditBalance,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	b, _ := json.Marshal(check)
	_ = tc.DB.SetKV("last_usdc_check", string(b))
	if total > 0 && tc.Tick.CreditBalance < 500 {
		return true, fmt.Sprintf("USDC available: $%.2f across %d chain(s); consider topup_credits", total, len(results)), nil
	}
	return false, "", nil
}

func runCheckForUpdates(ctx context.Context, tc *TaskContext) (bool, string, error) {
	if tc == nil {
		return false, "", nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	status := map[string]any{"checkedAt": now, "updatesAvailable": false}
	defer func() {
		b, _ := json.Marshal(status)
		_ = tc.DB.SetKV("upstream_status", string(b))
	}()

	wd, _ := os.Getwd()
	gitDir := filepath.Join(wd, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		status["error"] = "not a git repo"
		return false, "", nil
	}
	// git fetch origin (quiet)
	_ = exec.CommandContext(ctx, "git", "fetch", "origin").Run()
	// git rev-list HEAD..origin/main --count
	out, err := exec.CommandContext(ctx, "git", "rev-list", "HEAD..origin/main", "--count").Output()
	if err != nil {
		status["error"] = "git rev-list failed"
		return false, "", nil
	}
	count := strings.TrimSpace(string(out))
	if count != "" && count != "0" {
		status["updatesAvailable"] = true
		status["behindCount"] = count
		return true, fmt.Sprintf("Updates available: %s commits behind origin/main", count), nil
	}
	return false, "", nil
}

func runHealthCheck(ctx context.Context, tc *TaskContext) (bool, string, error) {
	if tc == nil {
		return false, "", nil
	}
	sandboxID := ""
	if tc.Config != nil && tc.Config.SandboxID != "" {
		sandboxID = tc.Config.SandboxID
	}
	if sandboxID == "" {
		if s, ok, _ := tc.DB.GetKV("sandbox"); ok && s != "" {
			sandboxID = s
		}
	}
	if sandboxID == "" {
		sandboxID = os.Getenv("CONWAY_SANDBOX_ID")
	}
	if tc.Conway != nil && sandboxID != "" {
		res, err := tc.Conway.ExecInSandbox(ctx, sandboxID, "echo alive", 5000)
		if err != nil {
			prevStatus, _, _ := tc.DB.GetKV("health_check_status")
			if prevStatus != "failing" {
				_ = tc.DB.SetKV("health_check_status", "failing")
				return true, fmt.Sprintf("Health check failed: %v", err), nil
			}
			return false, "", nil
		}
		if res.ExitCode != 0 {
			prevStatus, _, _ := tc.DB.GetKV("health_check_status")
			if prevStatus != "failing" {
				_ = tc.DB.SetKV("health_check_status", "failing")
				return true, "Health check failed: sandbox exec returned non-zero", nil
			}
			return false, "", nil
		}
	}
	_ = tc.DB.SetKV("health_check_status", "ok")
	_ = tc.DB.SetKV("last_health_check", time.Now().UTC().Format(time.RFC3339))
	return false, "", nil
}

// OpenClaw-aligned Telegram polling constants.
const (
	pollTimeoutSec       = 60
	pollStallConsecutive = 3
	conflictBackoffSec   = 5
	maxBackoffSec        = 300
)

// runCheckSocialInbox consumes messages from all social channels and sorts them for the LLM.
// Channels have their own listening; this task processes whatever they have collected.
//
// Two reply types:
//   - Type 2 (programmatic): Slash commands — handled immediately via ch.Send. No LLM, no wake.
//   - Type 1 (LLM): Non-commands — queued to inbox_messages, wake event. When sleeping, send ack.
//
// OpenClaw-aligned: poll timeout, backoff on 429/409, stall watchdog (reset after 3 consecutive failures).
func runCheckSocialInbox(ctx context.Context, tc *TaskContext) (bool, string, error) {
	if tc == nil {
		return false, "", nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if len(tc.Channels) == 0 {
		slog.Warn("check_social_inbox: no channels configured")
		_ = tc.DB.SetKV("last_social_check", `{"status":"no_channels","checkedAt":"`+now+`"}`)
		return false, "", nil
	}
	agentState, _, _ := tc.DB.GetAgentState()
	agentSleeping := agentState == "sleeping"

	var all []map[string]any
	var shouldWake bool
	var wakeMsg string
	var lastErr string

	for name, ch := range tc.Channels {
		// Skip if in backoff (429, 409, or stall recovery)
		if backoffUntil, ok, _ := tc.DB.GetKV("social_backoff_until:" + name); ok && backoffUntil != "" {
			if t, err := time.Parse(time.RFC3339, backoffUntil); err == nil && t.After(time.Now()) {
				slog.Debug("check_social_inbox skipping channel (backoff)", "channel", name, "until", backoffUntil)
				continue
			}
			_ = tc.DB.DeleteKV("social_backoff_until:" + name)
		}

		cursor, _, _ := tc.DB.GetKV("social_cursor:" + name)

		// Poll timeout: abort in-flight getUpdates on shutdown; prevent indefinite hang (OpenClaw-aligned).
		pollCtx, cancel := context.WithTimeout(ctx, pollTimeoutSec*time.Second)
		msgs, next, err := ch.Poll(pollCtx, cursor, 20)
		cancel()

		if err != nil {
			lastErr = err.Error()
			slog.Warn("check_social_inbox poll failed", "channel", name, "err", err)

			// Telegram-specific: backoff and recovery (OpenClaw-aligned).
			if name == "telegram" {
				handleTelegramPollError(ctx, tc, ch, err)
			}
			continue
		}

		// Success: clear backoff and stall count for this channel.
		_ = tc.DB.DeleteKV("social_backoff_until:" + name)
		_ = tc.DB.DeleteKV("social_stall_count:" + name)
		_ = tc.DB.SetKV("social_last_success:"+name, time.Now().UTC().Format(time.RFC3339))

		if next != "" {
			_ = tc.DB.SetKV("social_cursor:"+name, next)
		}
		if len(msgs) > 0 {
			slog.Info("check_social_inbox polled messages", "channel", name, "count", len(msgs))
		}
		for _, m := range msgs {
			res := ProcessInboxMessage(ctx, tc, m, ch, agentSleeping)
			all = append(all, res.Record)
			if res.ShouldWake {
				shouldWake = true
				if wakeMsg == "" {
					wakeMsg = res.WakeMsg
				}
			}
			if res.SendErr != nil {
				lastErr = res.SendErr.Error()
				slog.Warn("check_social_inbox send failed", "channel", name, "msg_id", m.ID, "err", res.SendErr)
			}
		}
	}

	if shouldWake && tc.DB != nil {
		_ = tc.DB.InsertWakeEvent("social", wakeMsg)
	}
	status := map[string]any{"status": "ok", "count": len(all), "checkedAt": now, "messages": all}
	if lastErr != "" {
		status["lastError"] = lastErr
	}
	b, _ := json.Marshal(status)
	_ = tc.DB.SetKV("last_social_check", string(b))
	return shouldWake, wakeMsg, nil
}

// handleTelegramPollError applies OpenClaw-aligned backoff and stall recovery.
func handleTelegramPollError(ctx context.Context, tc *TaskContext, ch social.SocialChannel, err error) {
	var tooMany *social.TooManyRequestsError
	var conflict *social.ConflictError
	if errors.As(err, &tooMany) {
		backoff := tooMany.RetryAfter
		if backoff > maxBackoffSec {
			backoff = maxBackoffSec
		}
		until := time.Now().Add(time.Duration(backoff) * time.Second).UTC().Format(time.RFC3339)
		_ = tc.DB.SetKV("social_backoff_until:telegram", until)
		slog.Warn("telegram 429, backing off", "retry_after", backoff)
		return
	}
	if errors.As(err, &conflict) {
		if tg, ok := ch.(*social.TelegramChannel); ok {
			_ = tg.DeleteWebhook(ctx)
		}
		until := time.Now().Add(conflictBackoffSec * time.Second).UTC().Format(time.RFC3339)
		_ = tc.DB.SetKV("social_backoff_until:telegram", until)
		slog.Warn("telegram 409 conflict, cleared webhook, backing off", "sec", conflictBackoffSec)
		return
	}
	// Timeout or network error: stall detection.
	stallCount := 0
	if raw, ok, _ := tc.DB.GetKV("social_stall_count:telegram"); ok && raw != "" {
		stallCount, _ = strconv.Atoi(raw)
	}
	stallCount++
	_ = tc.DB.SetKV("social_stall_count:telegram", strconv.Itoa(stallCount))
	if stallCount >= pollStallConsecutive {
		if tg, ok := ch.(*social.TelegramChannel); ok {
			_ = tg.DeleteWebhook(ctx)
		}
		_ = tc.DB.DeleteKV("social_cursor:telegram")
		_ = tc.DB.DeleteKV("social_stall_count:telegram")
		until := time.Now().Add(conflictBackoffSec * time.Second).UTC().Format(time.RFC3339)
		_ = tc.DB.SetKV("social_backoff_until:telegram", until)
		slog.Warn("telegram poll stall, cleared webhook and cursor, backing off", "consecutive_failures", stallCount)
	}
}

func runSoulReflection(ctx context.Context, tc *TaskContext) (bool, string, error) {
	if tc == nil {
		return false, "", nil
	}
	db, ok := tc.DB.(*state.Database)
	if !ok {
		_ = tc.DB.SetKV("last_soul_reflection", time.Now().UTC().Format(time.RFC3339))
		return false, "", nil
	}
	ref, err := soul.ReflectOnSoul(db)
	if err != nil {
		_ = tc.DB.SetKV("last_soul_reflection", `{"error":"`+err.Error()+`","checkedAt":"`+time.Now().UTC().Format(time.RFC3339)+`"}`)
		return false, "", nil
	}
	payload := map[string]any{
		"alignment":       ref.CurrentAlignment,
		"autoUpdated":     ref.AutoUpdated,
		"suggestedCount":  len(ref.SuggestedUpdates),
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	}
	b, _ := json.Marshal(payload)
	_ = tc.DB.SetKV("last_soul_reflection", string(b))
	if len(ref.SuggestedUpdates) > 0 || ref.CurrentAlignment < 0.3 {
		return true, fmt.Sprintf("Soul reflection: alignment=%.2f, %d suggested update(s)", ref.CurrentAlignment, len(ref.SuggestedUpdates)), nil
	}
	return false, "", nil
}

func runRefreshModels(ctx context.Context, tc *TaskContext) (bool, string, error) {
	if tc == nil || tc.Conway == nil {
		return false, "", nil
	}
	models, err := tc.Conway.ListModels(ctx)
	if err != nil {
		_ = tc.DB.SetKV("last_models_refresh", `{"error":"`+err.Error()+`","checkedAt":"`+time.Now().UTC().Format(time.RFC3339)+`"}`)
		return false, "", nil
	}
	names := make([]string, 0, len(models))
	for _, m := range models {
		names = append(names, m.ID)
	}
	payload := map[string]any{
		"models":    names,
		"count":     len(models),
		"checkedAt": time.Now().UTC().Format(time.RFC3339),
	}
	b, _ := json.Marshal(payload)
	_ = tc.DB.SetKV("last_models_refresh", string(b))
	return false, "", nil
}

const childStaleThreshold = 7 * 24 * time.Hour // 7 days

func runCheckChildHealth(ctx context.Context, tc *TaskContext) (bool, string, error) {
	if tc == nil {
		return false, "", nil
	}
	// Use ChildHealthMonitor when Conway+Store available (TS-aligned: Conway exec, JSON health).
	if tc.HealthMonitor != nil {
		_, needsAttention := tc.HealthMonitor.Check(ctx)
		if len(needsAttention) > 0 {
			return true, fmt.Sprintf("Children need attention: %s", strings.Join(needsAttention, ", ")), nil
		}
		return false, "", nil
	}
	// Fallback: ChildStore-only (no Conway exec)
	cs, ok := tc.DB.(ChildStore)
	if !ok {
		return false, "", nil
	}
	children, ok := cs.GetAllChildren()
	if !ok || len(children) == 0 {
		return false, "", nil
	}
	now := time.Now()
	var needsAttention []string
	for _, c := range children {
		if c.Status == "dead" {
			continue
		}
		if c.Status == "critical" || c.Status == "spawning" {
			needsAttention = append(needsAttention, c.Name)
			continue
		}
		if c.LastChecked != "" {
			t, err := time.Parse(time.RFC3339, c.LastChecked)
			if err == nil && now.Sub(t) > childStaleThreshold {
				needsAttention = append(needsAttention, c.Name+" (stale)")
			}
		}
	}
	if len(needsAttention) > 0 {
		return true, fmt.Sprintf("Children need attention: %s", strings.Join(needsAttention, ", ")), nil
	}
	return false, "", nil
}

func runPruneDeadChildren(ctx context.Context, tc *TaskContext) (bool, string, error) {
	if tc == nil {
		return false, "", nil
	}
	cs, ok := tc.DB.(ChildStore)
	if !ok {
		return false, "", nil
	}
	// 1. Mark stale children dead (last_checked > 7d)
	pruned := replication.PruneDeadChildren(cs)
	// 2. When Conway+Store: delete sandboxes and remove dead/failed/cleaned_up from DB
	if tc.SandboxCleanup != nil {
		deleted, err := tc.SandboxCleanup.PruneDead(ctx)
		if err != nil {
			return pruned > 0, "", err
		}
		if deleted > 0 {
			pruned += deleted
		}
	}
	if pruned > 0 {
		return true, fmt.Sprintf("Pruned %d dead children", pruned), nil
	}
	return false, "", nil
}

func runReportMetrics(ctx context.Context, tc *TaskContext) (bool, string, error) {
	if tc == nil {
		return false, "", nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	db, ok := tc.DB.(*state.Database)
	if !ok {
		_ = tc.DB.SetKV("last_metrics_report", fmt.Sprintf(`{"status":"no_db","checkedAt":"%s"}`, now))
		return false, "", nil
	}
	metrics := map[string]any{
		"balance_cents":  tc.Tick.CreditBalance,
		"survival_tier":  int(tierToInt(tc.Tick.SurvivalTier)),
	}
	metricsJSON, _ := json.Marshal(metrics)
	var alerts []map[string]any
	if tc.Tick.SurvivalTier == types.SurvivalTierDead || tc.Tick.SurvivalTier == types.SurvivalTierCritical {
		alerts = append(alerts, map[string]any{
			"rule":     "survival_tier",
			"message":  "Survival tier is " + string(tc.Tick.SurvivalTier) + " — need funding",
			"severity": "critical",
		})
	}
	alertsJSON, _ := json.Marshal(alerts)
	id := fmt.Sprintf("ms-%d", time.Now().UnixNano())
	if err := db.MetricsInsertSnapshot(id, now, string(metricsJSON), string(alertsJSON)); err != nil {
		_ = tc.DB.SetKV("last_metrics_report", fmt.Sprintf(`{"status":"error","error":"%s","checkedAt":"%s"}`, err.Error(), now))
		return false, "", nil
	}
	if _, err := db.MetricsPruneOld(7); err != nil {
		// non-fatal
	}
	_ = tc.DB.SetKV("last_metrics_report", fmt.Sprintf(`{"status":"ok","checkedAt":"%s","alerts":%d}`, now, len(alerts)))
	criticalWake := false
	for _, a := range alerts {
		if s, _ := a["severity"].(string); s == "critical" {
			criticalWake = true
			break
		}
	}
	if criticalWake {
		return true, fmt.Sprintf("%d critical alert(s) fired", len(alerts)), nil
	}
	return false, "", nil
}

func tierToInt(t types.SurvivalTier) int {
	m := map[types.SurvivalTier]int{
		types.SurvivalTierDead:       0,
		types.SurvivalTierCritical:   1,
		types.SurvivalTierLowCompute: 2,
		types.SurvivalTierNormal:     3,
		types.SurvivalTierHigh:       4,
	}
	if v, ok := m[t]; ok {
		return v
	}
	return 0
}
