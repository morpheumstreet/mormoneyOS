package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/morpheumlabs/mormoneyos-go/internal/config"
	"github.com/morpheumlabs/mormoneyos-go/internal/social"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
	_ "modernc.org/sqlite"
)

var testTelegramCmd = &cobra.Command{
	Use:   "test-telegram",
	Short: "Verify Telegram bot connectivity and message flow",
	Long: `Runs checks to verify the Telegram bot can:
  1. Connect (getMe, deleteWebhook)
  2. Poll for updates (getUpdates)
  3. Send messages (optional, with --send-to)

Use this to verify the message flow: Telegram network → mormoneyOS → Telegram.`,
	RunE: runTestTelegram,
}

func init() {
	rootCmd.AddCommand(testTelegramCmd)
	testTelegramCmd.Flags().String("send-to", "", "Chat ID to send a test message (e.g. your user ID or group ID)")
	testTelegramCmd.Flags().Bool("diag", false, "Show runtime diagnostic state (where Telegram flow might be stuck)")
}

func runTestTelegram(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg == nil {
		return fmt.Errorf("no config found; run 'moneyclaw init' first")
	}

	diag, _ := cmd.Flags().GetBool("diag")
	if diag {
		return runDiagTelegram(cfg)
	}

	if cfg.TelegramBotToken == "" {
		return fmt.Errorf("telegramBotToken not configured; add it via setup or config")
	}

	ch, err := social.NewTelegramChannel(cfg)
	if err != nil {
		return fmt.Errorf("create telegram channel: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. HealthCheck (getMe, deleteWebhook, setMyCommands)
	fmt.Print("1. HealthCheck (getMe, deleteWebhook)... ")
	if err := ch.HealthCheck(ctx); err != nil {
		fmt.Printf("FAIL — %v\n", err)
		return err
	}
	fmt.Println("OK")

	// 2. Poll (getUpdates)
	fmt.Print("2. Poll (getUpdates)... ")
	msgs, next, err := ch.Poll(ctx, "", 5)
	if err != nil {
		fmt.Printf("FAIL — %v\n", err)
		return err
	}
	fmt.Printf("OK (cursor=%s, messages=%d)\n", next, len(msgs))

	// 3. Optional: send test message
	sendTo, _ := cmd.Flags().GetString("send-to")
	if sendTo != "" {
		fmt.Printf("3. Send test message to %s... ", sendTo)
		msg := &social.OutboundMessage{
			Content:   "mormoneyOS test-telegram: connectivity check OK.",
			Recipient: sendTo,
		}
		if _, err := ch.Send(ctx, msg); err != nil {
			fmt.Printf("FAIL — %v\n", err)
			return err
		}
		fmt.Println("OK")
	} else {
		fmt.Println("3. Send (skipped; use --send-to CHAT_ID to test)")
	}

	fmt.Println("\nAll checks passed. Telegram message flow is ready.")
	return nil
}

func runDiagTelegram(cfg *types.AutomatonConfig) error {
	dbPath := config.ResolvePath(cfg.DBPath)
	if dbPath == "" {
		dbPath = config.GetAutomatonDir() + "/state.db"
	}
	db, err := state.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open db %s: %w", dbPath, err)
	}
	defer db.Close()

	fmt.Println("Telegram flow diagnostic (runtime state from DB)")
	fmt.Println("================================================")

	// KV state
	cursor, _, _ := db.GetKV("social_cursor:telegram")
	backoff, _, _ := db.GetKV("social_backoff_until:telegram")
	stall, _, _ := db.GetKV("social_stall_count:telegram")
	lastSuccess, _, _ := db.GetKV("social_last_success:telegram")
	lastCheck, _, _ := db.GetKV("last_social_check")
	agentState, _, _ := db.GetAgentState()

	fmt.Printf("\n1. Poll state:\n")
	fmt.Printf("   social_cursor:telegram   = %q\n", cursor)
	fmt.Printf("   social_backoff_until     = %q\n", backoff)
	fmt.Printf("   social_stall_count      = %q\n", stall)
	fmt.Printf("   social_last_success     = %q\n", lastSuccess)
	fmt.Printf("   last_social_check       = %s\n", truncate(lastCheck, 400))
	if lastCheck != "" {
		var check struct {
			LastError string `json:"lastError"`
			Count     int    `json:"count"`
		}
		_ = json.Unmarshal([]byte(lastCheck), &check)
		if check.LastError != "" {
			fmt.Printf("   >>> lastError (send failed) = %q\n", check.LastError)
		}
	}

	fmt.Printf("\n2. Agent:\n")
	fmt.Printf("   agent_state             = %q\n", agentState)

	// Heartbeat schedule for check_social_inbox
	rows, err := db.GetHeartbeatSchedule()
	if err != nil {
		fmt.Printf("\n3. Heartbeat schedule: (err: %v)\n", err)
	} else {
		fmt.Printf("\n3. Heartbeat schedule (check_social_inbox):\n")
		for _, r := range rows {
			if r.Name == "check_social_inbox" {
				fmt.Printf("   enabled=%d schedule=%q last_run=%q\n", r.Enabled, r.Schedule, r.LastRun)
				break
			}
		}
	}

	fmt.Printf("\n4. Tip: Run 'moneyclaw run' and watch logs for 'check_social_inbox' or poll errors.\n")

	// Diagnose likely stuck point
	fmt.Printf("\n5. Likely stuck point:\n")
	if backoff != "" {
		if t, err := time.Parse(time.RFC3339, backoff); err == nil && t.After(time.Now()) {
			fmt.Printf("   >>> IN BACKOFF until %s (429/409/stall recovery)\n", backoff)
			return nil
		}
	}
	if lastSuccess == "" && cursor == "" {
		fmt.Printf("   >>> Poll may never have succeeded; cursor empty\n")
		return nil
	}
	if lastCheck == "" {
		fmt.Printf("   >>> last_social_check empty — check_social_inbox may not be running\n")
		fmt.Printf("   >>> Run 'moneyclaw run' — the runtime must be active for Telegram polling.\n")
		return nil
	}
	// Check if runtime is likely running (last_run within last 2 min)
	if rows, err := db.GetHeartbeatSchedule(); err == nil {
		for _, r := range rows {
			if r.Name == "check_social_inbox" && r.LastRun != "" {
				if t, err := time.Parse(time.RFC3339, r.LastRun); err == nil {
					ago := time.Since(t)
					if ago > 2*time.Minute {
						fmt.Printf("   >>> Last check_social_inbox run was %v ago — is 'moneyclaw run' active?\n", ago.Round(time.Second))
					}
				}
				break
			}
		}
	}
	fmt.Printf("   If commands get no reply: check logs for 'social command send failed' or 'check_social_inbox send failed'.\n")
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
