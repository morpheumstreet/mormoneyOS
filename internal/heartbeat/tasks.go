package heartbeat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
		{Name: "check_social_inbox", Schedule: "*/15 * * * *", Run: runCheckSocialInbox},
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
	// Go: no USDC balance API yet; stub
	if tc == nil {
		return false, "", nil
	}
	credits := tc.Tick.CreditBalance
	tier := tc.Tick.SurvivalTier
	check := map[string]any{
		"balance": 0, "credits": credits, "timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	b, _ := json.Marshal(check)
	_ = tc.DB.SetKV("last_usdc_check", string(b))
	_ = tier
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
	// Go: no Conway exec yet; stub - just record ok
	if tc == nil {
		return false, "", nil
	}
	_ = tc.DB.SetKV("health_check_status", "ok")
	_ = tc.DB.SetKV("last_health_check", time.Now().UTC().Format(time.RFC3339))
	return false, "", nil
}

func runCheckSocialInbox(ctx context.Context, tc *TaskContext) (bool, string, error) {
	// Go: no social client yet; stub
	if tc == nil {
		return false, "", nil
	}
	_ = tc.DB.SetKV("last_social_check", `{"status":"stub","checkedAt":"`+time.Now().UTC().Format(time.RFC3339)+`"}`)
	return false, "", nil
}

func runSoulReflection(ctx context.Context, tc *TaskContext) (bool, string, error) {
	// Go: stub; TS soul_reflection triggers LLM reflection. Record last check.
	if tc == nil {
		return false, "", nil
	}
	_ = tc.DB.SetKV("last_soul_reflection", time.Now().UTC().Format(time.RFC3339))
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
	children, ok := cs.GetAllChildren()
	if !ok || len(children) == 0 {
		return false, "", nil
	}
	now := time.Now()
	pruned := 0
	for _, c := range children {
		if c.Status == "dead" {
			continue
		}
		if c.LastChecked == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, c.LastChecked)
		if err != nil {
			continue
		}
		if now.Sub(t) > childStaleThreshold {
			if err := cs.UpdateChildStatus(c.ID, "dead"); err == nil {
				pruned++
			}
		}
	}
	if pruned > 0 {
		return true, fmt.Sprintf("Pruned %d dead children", pruned), nil
	}
	return false, "", nil
}

func runReportMetrics(ctx context.Context, tc *TaskContext) (bool, string, error) {
	// Go: stub; TS report_metrics sends metrics to external endpoint.
	if tc == nil {
		return false, "", nil
	}
	_ = tc.DB.SetKV("last_metrics_report", fmt.Sprintf(`{"status":"stub","checkedAt":"%s"}`, time.Now().UTC().Format(time.RFC3339)))
	return false, "", nil
}
