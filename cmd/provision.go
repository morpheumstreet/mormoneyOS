package cmd

import (
	"fmt"
	"os"

	"github.com/morpheumlabs/mormoneyos-go/internal/identity"
	"github.com/spf13/cobra"
)

var provisionCmd = &cobra.Command{
	Use:   "provision",
	Short: "Provision Conway API key via SIWE",
	Long:  `Sign in with Ethereum to create and save a Conway API key. Requires wallet (run 'moneyclaw init' first).`,
	RunE:  runProvision,
}

func init() {
	provisionCmd.Flags().String("api-url", "https://api.conway.tech", "Conway API base URL")
	provisionCmd.Flags().String("chain", identity.DefaultChainBase, "CAIP-2 chain for SIWE (e.g. eip155:8453)")
}

func runProvision(cmd *cobra.Command, args []string) error {
	apiURL, _ := cmd.Flags().GetString("api-url")
	chain, _ := cmd.Flags().GetString("chain")
	result, err := identity.Provision(apiURL, chain)
	if err != nil {
		return fmt.Errorf("provision: %w", err)
	}
	fmt.Fprintf(os.Stderr, "API key provisioned: %s...\n", result.KeyPrefix)
	fmt.Println(result.APIKey)
	return nil
}
