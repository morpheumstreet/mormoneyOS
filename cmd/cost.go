package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var costCmd = &cobra.Command{
	Use:   "cost",
	Short: "Show LLM cost summary",
	Long:  `Show agent's own running cost (moneyclaw-py aligned). LLM cost tracker not yet implemented.`,
	RunE:  runCost,
}

func runCost(cmd *cobra.Command, args []string) error {
	// Placeholder: LLM cost tracker not yet implemented
	fmt.Fprintln(os.Stdout, "LLM Cost Summary (placeholder)")
	fmt.Fprintln(os.Stdout, "  Today:    $0.00")
	fmt.Fprintln(os.Stdout, "  Total:   $0.00")
	fmt.Fprintln(os.Stdout, "  Calls:   0")
	fmt.Fprintln(os.Stdout, "\nLLM cost tracker not yet implemented.")
	return nil
}
