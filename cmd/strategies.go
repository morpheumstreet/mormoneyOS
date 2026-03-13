package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var strategiesCmd = &cobra.Command{
	Use:   "strategies",
	Short: "List discovered strategies",
	Long:  `List strategy plugins (moneyclaw-py aligned). Strategy plugin system not yet implemented.`,
	RunE:  runStrategies,
}

func runStrategies(cmd *cobra.Command, args []string) error {
	// Placeholder: strategy plugin system not yet implemented
	strategies := []struct {
		name        string
		description string
		risk        string
	}{
		{"crypto_dca", "DCA into crypto — buy fixed amounts on schedule", "low"},
		{"crypto_price_alert", "Monitor crypto prices, alert on threshold crossings", "low"},
		{"crypto_funding", "Funding rate arbitrage — collect high perp funding fees", "medium"},
		{"stock_dividend", "Track high-dividend stocks, alert before ex-dividend", "low"},
		{"smart_rebalance", "Maintain target portfolio allocation with market-aware rebalancing", "medium"},
	}
	fmt.Fprintln(os.Stdout, "Discovered strategies (placeholder):")
	for _, s := range strategies {
		fmt.Fprintf(os.Stdout, "  %-20s %s [%s]\n", s.name, s.description, s.risk)
	}
	fmt.Fprintln(os.Stdout, "\nStrategy plugin system not yet implemented. Use web dashboard for status.")
	return nil
}
