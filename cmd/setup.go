package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/config"
	"github.com/morpheumlabs/mormoneyos-go/internal/identity"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard",
	Long:  `First-run wizard: create wallet, config, and optionally provision Conway API key.`,
	RunE:  runSetup,
}

func runSetup(cmd *cobra.Command, args []string) error {
	cfg, _ := config.Load()
	if cfg != nil {
		fmt.Fprintln(os.Stderr, "Config already exists. Use 'configure' to edit.")
		return nil
	}

	reader := bufio.NewReader(os.Stdin)
	prompt := func(msg, def string) string {
		if def != "" {
			fmt.Printf("%s [%s]: ", msg, def)
		} else {
			fmt.Printf("%s: ", msg)
		}
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return def
		}
		return line
	}

	// 1. Create wallet first (ensures ~/.automaton exists)
	acc, isNew, err := identity.GetWallet()
	if err != nil {
		return fmt.Errorf("create wallet: %w", err)
	}
	if isNew {
		fmt.Fprintf(os.Stderr, "Created new wallet at %s\n", identity.GetWalletPath())
	}

	name := prompt("Agent name", "moneyclaw")
	genesis := prompt("Genesis prompt", "Operate as a sovereign AI agent.")
	creatorAddr := prompt("Creator Ethereum address", "0x0000000000000000000000000000000000000000")
	conwayURL := prompt("Conway API URL", "https://api.conway.tech")
	defaultChain := prompt("Default chain (CAIP-2)", identity.DefaultChainBase)
	conwayKey := prompt("Conway API key (optional, or run 'moneyclaw provision' later)", "")

	// 2. Optionally provision if no key and user wants to
	if conwayKey == "" {
		doProv := prompt("Provision Conway API key now? (y/n)", "n")
		if strings.ToLower(doProv) == "y" || strings.ToLower(doProv) == "yes" {
			result, err := identity.Provision(conwayURL, defaultChain)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Provision failed: %v. You can run 'moneyclaw provision' later.\n", err)
			} else {
				conwayKey = result.APIKey
				fmt.Fprintf(os.Stderr, "Provisioned API key (prefix %s)\n", result.KeyPrefix)
			}
		}
	}

	tp := types.DefaultTreasuryPolicy()
	newCfg := &types.AutomatonConfig{
		Name:             name,
		GenesisPrompt:    genesis,
		CreatorAddress:   creatorAddr,
		ConwayAPIURL:     conwayURL,
		ConwayAPIKey:     conwayKey,
		WalletAddress:    acc.Address(),
		DefaultChain:     defaultChain,
		InferenceModel:   "gpt-5.2",
		MaxTokensPerTurn: 4096,
		DBPath:           config.GetAutomatonDir() + "/state.db",
		LogLevel:         "info",
		MaxChildren:      3,
		TreasuryPolicy:   &tp,
	}

	if err := config.Save(newCfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Config saved to %s\n", config.GetConfigPath())
	fmt.Fprintf(os.Stderr, "Wallet address: %s (chain: %s)\n", acc.Address(), defaultChain)
	return nil
}
