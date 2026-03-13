package cmd

import (
	"fmt"
	"os"

	"github.com/morpheumlabs/mormoneyos-go/internal/config"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show runtime status",
	Long:  `Display config path, database path, and basic status.`,
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg == nil {
		fmt.Fprintln(os.Stderr, "No config. Run 'moneyclaw setup' first.")
		return nil
	}

	fmt.Printf("Config:   %s\n", config.GetConfigPath())
	fmt.Printf("DB:       %s\n", cfg.DBPath)
	fmt.Printf("Name:     %s\n", cfg.Name)
	fmt.Printf("Conway:   %s\n", cfg.ConwayAPIURL)
	return nil
}
