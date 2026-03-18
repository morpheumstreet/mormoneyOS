// Package cli provides migration CLI commands.
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/morpheumlabs/mormoneyos-go/internal/config"
	"github.com/morpheumlabs/mormoneyos-go/internal/migration/converter"
	"github.com/morpheumlabs/mormoneyos-go/internal/migration/usecase"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// MigrateCmd is the one-click OpenClaw → MormOS migration command.
var MigrateCmd = &cobra.Command{
	Use:   "migrate [old-claw-path]",
	Short: "One-click OpenClaw → MormOS migration (SOUL.md + wallet + skills)",
	Long: `Migrate an OpenClaw agent to MormOS.

Reads SOUL.md and MEMORY.md from the given path, converts to SKILL.md v2 format,
and installs the skill into your MormOS runtime. Your existing agent becomes
Morpheum-native in minutes.

Example:
  moneyclaw migrate ./my-old-claw-agent
  moneyclaw migrate ~/openclaw-agents/trading-bot`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMigrate,
}

func init() {
	MigrateCmd.Flags().Bool("dry-run", false, "Show what would be migrated without writing")
}

func runMigrate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: moneyclaw migrate <old-claw-path>\nExample: moneyclaw migrate ./my-old-claw-agent")
	}
	oldPath := args[0]
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg == nil {
		return fmt.Errorf("no config: run 'moneyclaw setup' first")
	}

	dbPath := config.ResolvePath(cfg.DBPath)
	db, err := state.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	skillsCfg := cfg.Skills
	if skillsCfg == nil {
		skillsCfg = &types.SkillsConfig{}
	}
	uc := &usecase.MigrateAgent{
		Store:  db,
		Config: skillsCfg,
	}

	if dryRun {
		// Just validate conversion without writing
		_, err := converter.ConvertSoulToSkill(oldPath)
		if err != nil {
			return err
		}
		cmd.Println("✅ Dry run: SOUL.md + MEMORY.md would be converted successfully")
		cmd.Println("Run without --dry-run to complete migration")
		return nil
	}

	result, err := uc.Execute(context.Background(), oldPath)
	if err != nil {
		return err
	}

	cmd.Println("✅ Migration complete!")
	cmd.Printf("   Skill: %s\n", result.SkillName)
	cmd.Printf("   Path: %s\n", result.SkillPath)
	cmd.Println(result.Message)
	cmd.Println("\nFirst 1,000 claws get bonus MORM (as promised)")
	return nil
}
